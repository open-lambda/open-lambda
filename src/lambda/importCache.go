package lambda

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/sandbox"
)

type ImportCache struct {
	codeDirs    *common.DirMaker
	scratchDirs *common.DirMaker
	pkgPuller   *PackagePuller
	sbPool      sandbox.SandboxPool
	root        *ImportCacheNode
}

// a node in a tree of Zygotes
//
// This imposes a structure on what Zygotes are created, but there may
// be nodes not currently backed by a Zygote (e.g., due to eviction,
// Sandbox death, etc)
type ImportCacheNode struct {
	// from config file:
	Packages []string           `json:"packages""`
	Children []*ImportCacheNode `json:"children"`

	// backpointers based on Children structure
	parent *ImportCacheNode

	// Packages of all our ancestors
	indirectPackages []string

	// everything above does not change after init, and so doesn't
	// require lock protection.  All below is protected by the mutex

	mutex sync.Mutex
	sb    sandbox.Sandbox

	// Sandbox for this node of the tree (may be nil); codeDir
	// doesn't contain a lambda, but does contain a packages dir
	// linking to the packages in Packages and indirectPackages.
	// Lazily initialized when Sandbox is first needed.
	codeDir string

	// inferred from Packages (lazily initialized when Sandbox is
	// first needed)
	topLevelMods []string
}

type ZygoteReq struct {
	meta   *sandbox.SandboxMeta
	parent chan sandbox.Sandbox
}

func NewImportCache(codeDirs *common.DirMaker, scratchDirs *common.DirMaker, sizeMb int, pp *PackagePuller) (ic *ImportCache, err error) {
	cache := &ImportCache{
		codeDirs:    codeDirs,
		scratchDirs: scratchDirs,
		pkgPuller:   pp,
	}

	// a static tree of Zygotes may be specified by a file (if so, parse and init it)
	cache.root = &ImportCacheNode{}
	switch treeConf := common.Conf.Import_cache_tree.(type) {
	case string:
		if treeConf != "" {
			var b []byte
			if strings.HasPrefix(treeConf, "{") && strings.HasSuffix(treeConf, "}") {
				b = []byte(treeConf)
			} else {
				b, err = ioutil.ReadFile(treeConf)
				if err != nil {
					return nil, fmt.Errorf("could not open import tree file (%v): %v\n", treeConf, err.Error())
				}
			}

			if err := json.Unmarshal(b, cache.root); err != nil {
				return nil, fmt.Errorf("could parse import tree file (%v): %v\n", treeConf, err.Error())
			}
		}
	case map[string]interface{}:
		b, err := json.Marshal(treeConf)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, cache.root); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unexpected type for import_cache_tree setting: %T", treeConf)
	}

	// check and print tree
	if len(cache.root.Packages) > 0 {
		return nil, fmt.Errorf("root node in import cache may not import packages\n")
	}
	cache.recursiveInit(cache.root, []string{})
	log.Printf("Import Cache Tree:")
	cache.root.Dump(0)

	// import cache gets its own sandbox pool
	sbPool, err := sandbox.SandboxPoolFromConfig("import-cache", sizeMb)
	if err != nil {
		return nil, err
	}
	cache.sbPool = sbPool

	return cache, nil
}

func (cache *ImportCache) Cleanup() {
	cache.root.mutex.Lock()
	defer cache.root.mutex.Unlock()

	rootSb := cache.root.sb
	if rootSb != nil {
		// should recursively kill them all
		rootSb.Destroy()
	}
	cache.sbPool.Cleanup()
}

// 1. populate parent field of every struct
// 2. populate indirectPackages to contain the packages of every ancestor
func (cache *ImportCache) recursiveInit(node *ImportCacheNode, indirectPackages []string) {
	node.indirectPackages = indirectPackages
	for _, child := range node.Children {
		child.parent = node
		cache.recursiveInit(child, node.AllPackages())
	}
}

// (1) find Zygote and (2) use it to try creating a new Sandbox
func (cache *ImportCache) Create(childSandboxPool sandbox.SandboxPool, isLeaf bool, codeDir, scratchDir string, meta *sandbox.SandboxMeta) (sandbox.Sandbox, error) {
	node := cache.root.Lookup(meta.Installs)
	if node == nil {
		panic(fmt.Errorf("did not find Zygote; at least expected to find the root"))
	}
	log.Printf("Try using Zygote from <%v>", node)
	return cache.createChildSandboxFromNode(childSandboxPool, node, isLeaf, codeDir, scratchDir, meta)
}

// use getSandboxOfNode to create a Zygote for the node (creating one
// if necessary), then use that Zygote to create a new Sandbox.
//
// the new Sandbox may either be for a Zygote, or a leaf Sandbox
func (cache *ImportCache) createChildSandboxFromNode(
	childSandboxPool sandbox.SandboxPool, node *ImportCacheNode, isLeaf bool,
	codeDir, scratchDir string, meta *sandbox.SandboxMeta) (sandbox.Sandbox, error) {

	// try twice, restarting parent Sandbox if it fails the first time
	forceNew := false
	for i := 0; i < 2; i++ {
		zygoteSB, isNew, err := cache.getSandboxOfNode(node, forceNew)
		if err != nil {
			return nil, err
		}

		sb, err := childSandboxPool.Create(zygoteSB, isLeaf, codeDir, scratchDir, meta)

		// isNew is guaranteed to be true on 2nd iteration
		if err != sandbox.FORK_FAILED || isNew {
			return sb, err
		}

		forceNew = true
	}

	panic("'unreachable' code")
}

// recursively create a chain of sandboxes through the tree, from the
// root to this node, then return that Sandbox.
//
// this is always used to get Zygotes (never leaves)
func (cache *ImportCache) getSandboxOfNode(node *ImportCacheNode, forceNew bool) (sb sandbox.Sandbox, isNew bool, err error) {
	node.mutex.Lock()
	defer node.mutex.Unlock()

	// FAST PATH: we already have a previously created Sandbox we can use
	if node.sb != nil {
		if forceNew {
			node.sb.Destroy()
			node.sb = nil
		} else {
			return node.sb, false, nil
		}
	}

	// SLOW PATH: we need to create a new Zygote Sandbox (perhaps
	// even a chain of them back to the root Zygote)

	// populate codeDir/packages with deps, and record top-level mods)
	if node.codeDir == "" {
		codeDir := cache.codeDirs.Make("import-cache")
		defer func() {
			if err != nil {
				if err := os.RemoveAll(codeDir); err != nil {
					log.Printf("could not cleanup %s: %v", codeDir, err)
				}
			}
		}()

		if _, err := cache.pkgPuller.InstallRecursive(codeDir, node.AllPackages()); err != nil {
			return nil, false, err
		}

		topLevelMods := []string{}
		for _, name := range node.Packages {
			pkg, err := cache.pkgPuller.GetPkg(name)
			if err != nil {
				return nil, false, err
			}
			topLevelMods = append(topLevelMods, pkg.meta.TopLevel...)
		}

		node.codeDir = codeDir
		node.topLevelMods = topLevelMods
	}

	// POLICY: what modules should we pre-import?  Top-level of
	// pre-initialized packages is just one possibility...
	meta := &sandbox.SandboxMeta{
		Installs: node.AllPackages(),
		Imports:  node.topLevelMods,
	}

	scratchDir := cache.scratchDirs.Make("import-cache")
	if node.parent != nil {
		sb, err = cache.createChildSandboxFromNode(cache.sbPool, node.parent, false, node.codeDir, scratchDir, meta)
	} else {
		sb, err = cache.sbPool.Create(nil, false, node.codeDir, scratchDir, meta)
	}

	node.sb = sb
	return sb, true, nil
}

// return concatenation of direct (.Packages) and indirect (.indirectPackages)
func (node *ImportCacheNode) AllPackages() []string {
	n := len(node.indirectPackages)
	return append(node.indirectPackages[:n:n], node.Packages...)
}

func (node *ImportCacheNode) Lookup(packages []string) *ImportCacheNode {
	// if this node imports a package that's not wanted by the
	// lambda, neither this Zygote nor its children will work
	for _, nodePkg := range node.Packages {
		found := false
		for _, p := range packages {
			if p == nodePkg {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	// check our descendents; is one of them a Zygote that works?
	// we prefer a child Zygote over the one for this node,
	// because they have more packages pre-imported
	for _, child := range node.Children {
		result := child.Lookup(packages)
		if result != nil {
			return result
		}
	}

	return node
}

func (node *ImportCacheNode) String() string {
	s := strings.Join(node.Packages, ",")
	if s == "" {
		s = "ROOT"
	}
	if len(node.indirectPackages) > 0 {
		s += " [indirect: " + strings.Join(node.indirectPackages, ",") + "]"
	}
	return s
}

func (node *ImportCacheNode) Dump(indent int) {
	spaces := strings.Repeat("  ", indent)

	log.Printf("%s - %s", spaces, node.String())
	for _, child := range node.Children {
		child.Dump(indent + 1)
	}
}

package zygote

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/lambda/packages"
	"github.com/open-lambda/open-lambda/ol/worker/sandbox"
)

type ImportCache struct {
	codeDirs    *common.DirMaker
	scratchDirs *common.DirMaker
	pkgPuller   *packages.PackagePuller
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
	Packages []string           `json:"packages"`
	Children []*ImportCacheNode `json:"children"`

	// backpointers based on Children structure
	parent *ImportCacheNode

	// Packages of all our ancestors
	indirectPackages []string

	// everything above does not change after init, and so doesn't
	// require lock protection.  All below is protected by the mutex

	mutex      sync.Mutex
	sb         sandbox.Sandbox
	sbRefCount int // sb will be unpaused iff this is >0

	// create stats
	createNonleafChild int64
	createLeafChild    int64

	// Sandbox for this node of the tree (may be nil); codeDir
	// doesn't contain a lambda, but does contain a packages dir
	// linking to the packages in Packages and indirectPackages.
	// Lazily initialized when Sandbox is first needed.
	codeDir string

	// inferred from Packages (lazily initialized when Sandbox is
	// first needed)
	meta *sandbox.SandboxMeta
}

type ZygoteReq struct {
	parent chan sandbox.Sandbox
}

// NewImportCache creates a new ImportCache instance and initializes it with the given parameters.
func NewImportCache(codeDirs *common.DirMaker, scratchDirs *common.DirMaker, sbPool sandbox.SandboxPool, pp *packages.PackagePuller) (ic *ImportCache, err error) {
	cache := &ImportCache{
		codeDirs:    codeDirs,
		scratchDirs: scratchDirs,
		sbPool:      sbPool,
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
	case map[string]any:
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

	return cache, nil
}

// Cleanup performs cleanup operations for the ImportCache and its nodes.
func (cache *ImportCache) Cleanup() {
	log.Printf("Import Cache Tree:")
	cache.root.Dump(0)
	cache.recursiveKill(cache.root)
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

func (cache *ImportCache) recursiveKill(node *ImportCacheNode) {
	for _, child := range node.Children {
		cache.recursiveKill(child)
	}

	node.mutex.Lock()
	if node.sb != nil {
		node.sb.Destroy("ImportCache recursiveKill")
		node.sb = nil
	}
	node.mutex.Unlock()
}

// Create creates a new sandbox using the import cache.
func (cache *ImportCache) Create(childSandboxPool sandbox.SandboxPool, isLeaf bool, codeDir, scratchDir string, meta *sandbox.SandboxMeta, rt_type common.RuntimeType) (sandbox.Sandbox, error) {
	t := common.T0("ImportCache.Create")
	defer t.T1()

	t2 := common.T0("ImportCache.root.Lookup")
	node := cache.root.Lookup(meta.Installs)
	t2.T1()

	if node == nil {
		panic(fmt.Errorf("did not find Zygote; at least expected to find the root"))
	}
	log.Printf("Try using Zygote from <%v>", node)
	return cache.createChildSandboxFromNode(childSandboxPool, node, isLeaf, codeDir, scratchDir, meta, rt_type)
}

// use getSandboxInNode to get a Zygote Sandbox for the node (creating one
// if necessary), then use that Zygote Sandbox to create a new Sandbox.
//
// the new Sandbox may either be for a Zygote, or a leaf Sandbox
func (cache *ImportCache) createChildSandboxFromNode(
	childSandboxPool sandbox.SandboxPool, node *ImportCacheNode, isLeaf bool,
	codeDir, scratchDir string, meta *sandbox.SandboxMeta, rt_type common.RuntimeType) (sandbox.Sandbox, error) {
	t := common.T0("ImportCache.createChildSandboxFromNode")
	defer t.T1()

	// try twice, restarting parent Sandbox if it fails the first time
	forceNew := false
	for i := 0; i < 2; i++ {
		zygoteSB, isNew, err := cache.getSandboxInNode(node, forceNew, rt_type)
		if err != nil {
			return nil, err
		}

		t2 := common.T0("ImportCache.createChildSandboxFromNode:childSandboxPool.Create")
		sb, err := childSandboxPool.Create(zygoteSB, isLeaf, codeDir, scratchDir, meta, rt_type)

		if err == nil {
			if isLeaf {
				atomic.AddInt64(&node.createLeafChild, 1)
			} else {
				atomic.AddInt64(&node.createNonleafChild, 1)
			}
		}
		t2.T1()

		// dec ref count
		cache.putSandboxInNode(node, zygoteSB)

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
//
// the Sandbox returned is guaranteed to be in Unpaused state.  After
// use, caller must also call putSandboxInNode to release ref count
func (cache *ImportCache) getSandboxInNode(node *ImportCacheNode, forceNew bool, rt_type common.RuntimeType) (sb sandbox.Sandbox, isNew bool, err error) {
	t := common.T0("ImportCache.getSandboxInNode")
	defer t.T1()

	node.mutex.Lock()
	defer node.mutex.Unlock()

	// destroy any old Sandbox first if we're required to do so
	if forceNew && node.sb != nil {
		old := node.sb
		node.sb = nil
		go old.Destroy("ImportCache forceNew used")
	}

	if node.sb != nil {
		// FAST PATH
		if node.sbRefCount == 0 {
			if err := node.sb.Unpause(); err != nil {
				node.sb = nil
				return nil, false, err
			}
		}
		node.sbRefCount += 1
		return node.sb, false, nil
	}

	// SLOW PATH
	if err := cache.createSandboxInNode(node, rt_type); err != nil {
		return nil, false, err
	}
	node.sbRefCount = 1
	return node.sb, true, nil
}

// decrease refs to SB, pausing if nobody else is still using it
func (*ImportCache) putSandboxInNode(node *ImportCacheNode, sb sandbox.Sandbox) {
	t := common.T0("ImportCache.putSandboxInNode")
	defer t.T1()

	t2 := common.T0("ImportCache.putSandboxInNode:Lock")
	node.mutex.Lock()
	t2.T1()
	defer node.mutex.Unlock()

	if node.sb != sb {
		// the Sandbox must have been replaced (e.g., because
		// the Zygote fork failed to create another SB) after
		// we took a reference to it.  This means it is
		// already being destroyed, and we don't need to worry
		// about tracking references and pausing when we're
		// done
		return
	}

	node.sbRefCount -= 1

	if node.sbRefCount == 0 {
		t2 := common.T0("ImportCache.putSandboxInNode:Pause")
		if err := node.sb.Pause(); err != nil {
			node.sb = nil
		}
		t2.T1()
	}

	if node.sbRefCount < 0 {
		panic("negative ref count")
	}
}

func (cache *ImportCache) createSandboxInNode(node *ImportCacheNode, rt_type common.RuntimeType) (err error) {
	// populate codeDir/packages with deps, and record top-level mods)
	if node.codeDir == "" {
		codeDir := cache.codeDirs.Make("import-cache")
		// TODO: clean this up upon failure

		installs, err := cache.pkgPuller.InstallRecursive(node.Packages)
		if err != nil {
			return err
		}

		topLevelMods := []string{}
		for _, name := range node.Packages {
			pkg, err := cache.pkgPuller.GetPkg(name)
			if err != nil {
				return err
			}
			topLevelMods = append(topLevelMods, pkg.Meta.TopLevel...)
		}

		node.codeDir = codeDir

		// policy: what modules should we pre-import?  Top-level of
		// pre-initialized packages is just one possibility...
		node.meta = &sandbox.SandboxMeta{
			Installs: installs,
			Imports:  topLevelMods,
		}
	}

	scratchDir := cache.scratchDirs.Make("import-cache")
	var sb sandbox.Sandbox
	if node.parent != nil {
		sb, err = cache.createChildSandboxFromNode(cache.sbPool, node.parent, false, node.codeDir, scratchDir, node.meta, rt_type)
	} else {
		sb, err = cache.sbPool.Create(nil, false, node.codeDir, scratchDir, node.meta, common.RT_PYTHON)
	}

	if err != nil {
		return err
	}

	node.sb = sb
	return nil
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
	childCreates := fmt.Sprintf("%d", atomic.LoadInt64(&node.createLeafChild)+atomic.LoadInt64(&node.createNonleafChild))
	spaces := strings.Repeat(" ", indent*2+common.Max(0, 4-len(childCreates)))

	log.Printf("%s%s - %s", childCreates, spaces, node.String())
	for _, child := range node.Children {
		child.Dump(indent + 1)
	}
}

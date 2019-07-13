package lambda

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/sandbox"
)

type ImportCache struct {
	codeDirs *common.DirMaker
	scratchDirs *common.DirMaker
	pkgPuller *PackagePuller
	pool      sandbox.SandboxPool
	root      *ImportCacheNode
	requests  chan *ZygoteReq
	events    chan sandbox.SandboxEvent
	killChan  chan chan bool
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

	// Sandbox for this node of the tree (may be nil); codeDir
	// doesn't contain a lambda, but does contain a packages dir
	// linking to the packages in Packages and indirectPackages.
	// Lazily initialized when Sandbox is first needed.
	codeDir string
	sb      sandbox.Sandbox

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
		codeDirs: codeDirs,
		scratchDirs: scratchDirs,
		pkgPuller: pp,
		requests:  make(chan *ZygoteReq, 32),
		events:    make(chan sandbox.SandboxEvent, 32),
		killChan:  make(chan chan bool),
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
	pool, err := sandbox.SandboxPoolFromConfig("import-cache", sizeMb)
	if err != nil {
		return nil, err
	}
	pool.AddListener(cache.Event)
	cache.pool = pool

	// start background task to serve requests for Zygotes
	go cache.run(pool)
	return cache, nil
}

func (cache *ImportCache) Event(evType sandbox.SandboxEventType, sb sandbox.Sandbox) {
	if evType == sandbox.EvDestroy {
		cache.events <- sandbox.SandboxEvent{evType, sb}
	}
}

func (cache *ImportCache) Cleanup() {
	done := make(chan bool)
	cache.killChan <- done
	<-done
}

func (cache *ImportCache) GetZygote(meta *sandbox.SandboxMeta) sandbox.Sandbox {
	parent := make(chan sandbox.Sandbox)
	cache.requests <- &ZygoteReq{meta, parent}
	return <-parent
}

func (cache *ImportCache) run(pool sandbox.SandboxPool) {
	for {
		select {
		case req := <-cache.requests:
			// POLICY: which parent should we return?

			node := cache.root.Lookup(req.meta.Installs)
			if node == nil {
				panic(fmt.Errorf("did not find Zygote; at least expected to find the root"))
			}
			log.Printf("Try using Zygote from <%v>", node)

			sb, err := cache.getZygoteFromTree(node)
			if err != nil {
				log.Printf("getZygoteFromTree returned error: %v", err)
				sb = nil
			}

			req.parent <- sb

		case event := <-cache.events:
			// TODO: make sure we restart these as needed
			switch event.EvType {
			case sandbox.EvDestroy:
				log.Printf("Sandbox %v in import cache has been destroyed", event.SB.ID())
			}

		case done := <-cache.killChan:
			if cache.root.sb != nil {
				// should recursively kill them all
				cache.root.sb.Destroy()
			}
			cache.pool.Cleanup()
			done <- true
			return
		}
	}
}

// recursively create a chain of sandboxes through the tree, from the
// root to this node, then return that Sandbox
//
// if we want to parallelize the importCache, this is the function
// where we need to be careful with locking (searching for a Node in
// .Lookup(...) doesn't touch any data structs that change, so that
// could be done in parallel).
func (cache *ImportCache) getZygoteFromTree(node *ImportCacheNode) (sb sandbox.Sandbox, err error) {
	if node.sb != nil {
		return node.sb, nil
	}

	// we ONLY try if the parent Zygote we try forking from is
	// dead, and then we only retry at most once
	retry := true
Retry:

	// SandboxPool.Create params: (1) parent, (2) isLeaf [false for Zygotes], (3) codeDir, (4) scratchDir, (5) meta

	// (1) parent (recursively build the chain of ancestors to get a parent)
	var parentSB sandbox.Sandbox = nil
	if node.parent != nil {
		parentSB, err = cache.getZygoteFromTree(node.parent)
		if err != nil {
			return nil, err
		}
	}

	// (3) codeDir (populate codeDir/packages with deps, and record top-level mods)
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
			return nil, err
		}

		topLevelMods := []string{}
		for _, name := range node.Packages {
			pkg, err := cache.pkgPuller.GetPkg(name)
			if err != nil {
				return nil, err
			}
			topLevelMods = append(topLevelMods, pkg.meta.TopLevel...)
		}

		node.codeDir = codeDir
		node.topLevelMods = topLevelMods
	}

	// (4) scratchDir
	scratchDir := cache.scratchDirs.Make("import-cache")

	// (5) meta
	//
	// POLICY: what modules should we pre-import?  Top-level of
	// pre-initialized packages is just one possibility...
	meta := &sandbox.SandboxMeta{
		Installs: node.AllPackages(),
		Imports:  node.topLevelMods,
	}

	sb, err = cache.pool.Create(parentSB, false, node.codeDir, scratchDir, meta)
	if err != nil {
		// if there was a problem with the parent, we'll restart it and retry once more
		if err == sandbox.FORK_FAILED && retry {
			node.parent.sb.Destroy()
			node.parent.sb = nil
			retry = false
			goto Retry
		}

		return nil, err
	}
	node.sb = sb
	return sb, nil
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

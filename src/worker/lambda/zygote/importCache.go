package zygote

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
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
	Packages        []string           `json:"packages"`
	Children        []*ImportCacheNode `json:"children"`
	SplitGeneration int                `json:"split_generation"`

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

// (1) find Zygote and (2) use it to try creating a new Sandbox
func (cache *ImportCache) Create(childSandboxPool sandbox.SandboxPool, isLeaf bool, codeDir, scratchDir string,
	meta *sandbox.SandboxMeta, rt_type common.RuntimeType) (sandbox.Sandbox, int, error) {
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
	codeDir, scratchDir string, meta *sandbox.SandboxMeta, rt_type common.RuntimeType) (sandbox.Sandbox, int, error) {

	t := common.T0("ImportCache.createChildSandboxFromNode")
	defer t.T1()

	if !common.Conf.Features.COW {
		if isLeaf {
			sb, err := childSandboxPool.Create(node.sb, isLeaf, codeDir, scratchDir, meta, rt_type)
			return sb, 0, err
		} else {
			if node.sb != nil {
				return node.sb, 0, nil
			}
			sb, err := childSandboxPool.Create(nil, false, codeDir, scratchDir, meta, rt_type)
			return sb, 0, err
		}
	}
	// try twice, restarting parent Sandbox if it fails the first time
	forceNew := false
	for i := 0; i < 2; i++ {
		if forceNew {
			fmt.Printf("forceNew is true\n")
		}
		zygoteSB, isNew, miss, err := cache.getSandboxInNode(node, forceNew, rt_type, common.Conf.Features.COW)
		if err != nil {
			return nil, 0, err
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

		if isLeaf && sb != nil {
			sb.(*sandbox.SafeSandbox).Sandbox.(*sandbox.SOCKContainer).Node = node.SplitGeneration
		}

		// isNew is guaranteed to be true on 2nd iteration
		if err != sandbox.FORK_FAILED || isNew {
			return sb, miss, err
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
func (cache *ImportCache) getSandboxInNode(node *ImportCacheNode, forceNew bool, rt_type common.RuntimeType, cow bool,
) (sb sandbox.Sandbox, isNew bool, miss int, err error) {
	t := common.T0("ImportCache.getSandboxInNode")
	defer t.T1()

	t1 := common.T0("ImportCache.getSandboxInNode:Lock")
	node.mutex.Lock()
	t1.T1()
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
				return nil, false, 0, err
			}
		}
		node.sbRefCount += 1
		fmt.Printf("node.sb != nil: node %d, getSandboxInNode with ref count %d\n", node.SplitGeneration, node.sbRefCount)
		return node.sb, false, 0, nil
	} else {
		// SLOW PATH, miss >= 1
		if miss, err = cache.createSandboxInNode(node, rt_type, cow); err != nil {
			fmt.Printf("getSandboxInNode error: %s \n", err.Error())
			if node.parent != nil {
				fmt.Printf("node %d, parent %d\n", err.Error(), node.SplitGeneration, node.parent.SplitGeneration)
			}
			return nil, false, 0, err
		}
		node.sbRefCount = 1
		return node.sb, true, miss, nil
	}
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

var countMapLock sync.Mutex
var CreateCount = make(map[int]int)

func appendUnique(original []string, elementsToAdd []string) []string {
	exists := make(map[string]bool)
	for _, item := range original {
		exists[item] = true
	}

	for _, item := range elementsToAdd {
		if !exists[item] {
			original = append(original, item)
			exists[item] = true
		}
	}

	return original
}

// inherit the meta for all the ancestors
func inheritMeta(node *ImportCacheNode) (meta *sandbox.SandboxMeta) {
	tmpNode := node.parent
	meta = node.meta
	for tmpNode.SplitGeneration != 0 {
		if tmpNode.meta != nil {
			// merge meta
			meta.Imports = appendUnique(meta.Imports, tmpNode.meta.Imports)
		}
		tmpNode = tmpNode.parent
	}
	return meta
}

func (cache *ImportCache) createSandboxInNode(node *ImportCacheNode, rt_type common.RuntimeType, cow bool) (miss int, err error) {
	// populate codeDir/packages with deps, and record top-level mods)
	if node.codeDir == "" {
		codeDir := cache.codeDirs.Make("import-cache")
		// TODO: clean this up upon failure
		// todo: only thing is capture top-level mods, no need to open another sandbox
		// if all pkgs required by lambda are guaranteed to be installed, then no need to call getPkg(),
		// but sometimes a zygote is created without requests, e.g. warm up the tree, then getPkg() is needed
		installs := []string{}
		for _, name := range node.AllPackages() {
			_, err := cache.pkgPuller.GetPkg(name)
			if err != nil {
				return 0, fmt.Errorf("ImportCache.go: could not get package %s: %v", name, err)
			}
			installs = append(installs, name)
		}

		topLevelMods := []string{}
		for _, name := range node.Packages {
			pkgPath := filepath.Join(common.Conf.SOCK_base_path, "packages", name, "files")
			moduleInfos, err := packages.IterModules(pkgPath)
			if err != nil {
				return 0, err
			}
			modulesNames := []string{}
			for _, moduleInfo := range moduleInfos {
				modulesNames = append(modulesNames, moduleInfo.Name)
			}
			topLevelMods = append(topLevelMods, modulesNames...)
		}

		node.codeDir = codeDir

		// policy: what modules should we pre-import?  Top-level of
		// pre-initialized packages is just one possibility...
		node.meta = &sandbox.SandboxMeta{
			Installs:        installs,
			Imports:         topLevelMods,
			SplitGeneration: node.SplitGeneration,
		}
	}

	scratchDir := cache.scratchDirs.Make("import-cache")
	var sb sandbox.Sandbox
	miss = 0
	if node.parent != nil {
		if cow {
			sb, miss, err = cache.createChildSandboxFromNode(cache.sbPool, node.parent, false, node.codeDir, scratchDir, node.meta, rt_type)
		} else {
			node.meta = inheritMeta(node)
			// create a new sandbox without parent
			sb, err = cache.sbPool.Create(nil, false, node.codeDir, scratchDir, node.meta, common.RT_PYTHON)
		}
	} else {
		sb, err = cache.sbPool.Create(nil, false, node.codeDir, scratchDir, node.meta, common.RT_PYTHON)
	}

	if err != nil {
		return 0, err
	}

	node.sb = sb

	countMapLock.Lock()
	CreateCount[node.SplitGeneration] += 1
	countMapLock.Unlock()

	return miss + 1, nil
}

// Warmup will initialize every node in the tree,
// to have an accurate memory usage result and prevent warmup from failing, please have a large enough memory to avoid evicting
func (cache *ImportCache) Warmup() error {
	COW := common.Conf.Features.COW
	rt_type := common.RT_PYTHON

	warmupPy := "pass"
	// find all the leaf zygotes in the tree
	warmupZygotes := []*ImportCacheNode{}
	// do a BFS to find all the leaf zygote
	tmpNodes := []*ImportCacheNode{cache.root}

	// when COW is enabled, only create leaf zygotes(so that its parent will also be created)
	// when COW is disabled, create all zygotes
	for len(tmpNodes) > 0 {
		node := tmpNodes[0]
		tmpNodes = tmpNodes[1:]
		if !COW || len(node.Children) == 0 {
			warmupZygotes = append(warmupZygotes, node)
		}
		if len(node.Children) != 0 {
			tmpNodes = append(tmpNodes, node.Children...)
		}
	}

	errChan := make(chan error, len(warmupZygotes))
	var wg sync.WaitGroup

	goroutinePool := make(chan struct{}, 6)

	for i, node := range warmupZygotes {
		wg.Add(1)
		goroutinePool <- struct{}{}

		go func(i int, node *ImportCacheNode) {
			defer wg.Done()
			for _, pkg := range node.Packages {
				if _, err := cache.pkgPuller.GetPkg(pkg); err != nil {
					errChan <- fmt.Errorf("warmup: could not get package %s: %v", pkg, err)
					return
				}
			}

			zygoteSB, _, _, err := cache.getSandboxInNode(node, false, rt_type, COW)
			// if a created zygote is evicted in warmup, then node.sbRefCount will be 0
			if node.sbRefCount == 0 {
				fmt.Printf("warning: node %d has a refcnt %d<0, meaning it's destroyed\n", node.SplitGeneration, node.sbRefCount)
			}
			codeDir := cache.codeDirs.Make("warmup")
			// write warmyp_py to codeDir
			codePath := filepath.Join(codeDir, "f.py")
			ioutil.WriteFile(codePath, []byte(warmupPy), 0777)
			scratchDir := cache.scratchDirs.Make("warmup")
			sb, err := cache.sbPool.Create(zygoteSB, true, codeDir, scratchDir, nil, rt_type)
			if err != nil {
				errChan <- fmt.Errorf("failed to warm up zygote tree, reason is %s", err.Error())
				return
			}
			sb.Destroy("ensure modules are imported in ZygoteSB by launching a fork")
			atomic.AddInt64(&node.createNonleafChild, 1)
			cache.putSandboxInNode(node, zygoteSB)
			if err != nil {
				errChan <- fmt.Errorf("failed to warm up zygote tree, reason is %s", err.Error())
			} else {
				errChan <- nil
			}
			<-goroutinePool
		}(i, node)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return err
		}
	}

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

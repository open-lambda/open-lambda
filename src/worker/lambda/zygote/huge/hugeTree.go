package huge

import (
	"log"
	"sync"
	"path/filepath"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/lambda/packages"
	"github.com/open-lambda/open-lambda/ol/worker/sandbox"
)

type HugeTree struct {
	root *Node

	// the indexes between nodes and zygoteSets align
	nodes []*Node
	zygoteSets []*ZygoteSet

	// subsystems needed for creating sandboxes
	codeDirs    *common.DirMaker
	scratchDirs *common.DirMaker
	sbPool      sandbox.SandboxPool
	pkgPuller   *packages.PackagePuller
}

type Zygote struct {
	containingSet *ZygoteSet
	inUse         bool
	sb            sandbox.Sandbox
}

type ZygoteSet struct {
	mutex      sync.Mutex
	zygotes []*Zygote

	// all sandboxes in the save ZygoteSet will share a codedir
	// and metadata.
	codeDir string
	meta *sandbox.SandboxMeta
}

type CreateArgs struct {
	isLeaf bool
	codeDir string
	scratchDir string
	meta *sandbox.SandboxMeta
	rt_type common.RuntimeType
}

func NewHugeTree(
	codeDirs *common.DirMaker,
	scratchDirs *common.DirMaker,
	sbPool sandbox.SandboxPool,
	pp *packages.PackagePuller) (*HugeTree, error) {

	nodes, err := LoadTreeFromConfig()
	if err != nil {
		return nil, err
	}

	tree := &HugeTree{
		root: nodes[0],
		nodes: nodes,
		zygoteSets: []*ZygoteSet{},
		codeDirs:    codeDirs,
		scratchDirs: scratchDirs,
		sbPool:      sbPool,
		pkgPuller:   pp,
	}

	for i, node := range nodes {
		zygoteSet := &ZygoteSet{}
		if err := tree.initCodeDirIfNecessary(zygoteSet, node); err != nil {
			log.Printf("ZygoteSet %d/%d init FAILED for packages: %v\n", (i+1), len(nodes), node.Packages)
		} else {
			log.Printf("ZygoteSet %d/%d inititialized for packages: %v\n", (i+1), len(nodes), node.Packages)
		}
		tree.zygoteSets = append(tree.zygoteSets, zygoteSet)
	}

	return tree, nil
}

// initCodeDirIfNecessary creates a code dir to be used by all Zygotes
// in the set if it has not already been created.  The first-time init
// of this code dir is the only time set.mutex should be held for any
// significant amount of time.  This function assumes set.mutex is
// already held.
func (tree *HugeTree) initCodeDirIfNecessary(set *ZygoteSet, node *Node) error {
	if set.codeDir != "" {
		return nil
	}
	
	codeDir := tree.codeDirs.Make("import-cache")
	// TODO: clean this up upon failure

	// copied from metrics branch...
	topLevelMods := []string{}
	for _, name := range node.Packages {
		pkgPath := filepath.Join(common.Conf.SOCK_base_path, "packages", name, "files")
		moduleInfos, err := packages.IterModules(pkgPath)
		if err != nil {
			return err
		}
		modulesNames := []string{}
		for _, moduleInfo := range moduleInfos {
			modulesNames = append(modulesNames, moduleInfo.Name)
		}
		topLevelMods = append(topLevelMods, modulesNames...)
	}

	// policy: what modules should we pre-import?  Top-level of
	// pre-initialized packages is just one possibility...
	set.meta = &sandbox.SandboxMeta{
		Installs: node.AllPackages(),
		Imports:  topLevelMods,
	}

	log.Printf("Top Level: %v\n", topLevelMods)

	set.codeDir = codeDir
	return nil
}

// reserve tries to give us a Zygote.  If sbMustExist is true, reserve
// will either return nil, or a Zygote with a sandbox.  If sbMustExist
// is false, we are guaranteed to return a Zygote, but it may or may
// not contain a Sandbox.
//
// assume ZygoteSet lock is already held
func (set *ZygoteSet) reserve(sbMustExist bool) (*Zygote) {
	// any free zygotes with sandboxes already created?
	for _, zygote := range set.zygotes {
		if !zygote.inUse && zygote.sb != nil {
			zygote.inUse = true
			return zygote
		}
	}

	if sbMustExist {
		// we can't satisfy this
		return nil
	}

	// any free zygotes without sandboxes?
	for _, zygote := range set.zygotes {
		if !zygote.inUse {
			zygote.inUse = true
			return zygote
		}
	}

	// all are in use, so create a new one (which won't have a sandbox yet)
	zygote := &Zygote{containingSet: set, inUse: true}
	set.zygotes = append(set.zygotes, zygote)
	return zygote
}

func (zygote *Zygote) release() {
	zygote.containingSet.mutex.Lock()
	defer zygote.containingSet.mutex.Unlock()
	zygote.inUse = false
}

func (tree *HugeTree) Create(
	childSandboxPool sandbox.SandboxPool, isLeaf bool,
	codeDir, scratchDir string, meta *sandbox.SandboxMeta,
	rt_type common.RuntimeType) (sandbox.Sandbox, error) {

	args := CreateArgs{
		isLeaf: isLeaf,
		codeDir: codeDir,
		scratchDir: scratchDir,
		meta: meta,
		rt_type: rt_type,
	}
	// TODO: implement retry here
	sb, err := tree.tryCreate(childSandboxPool, args)
	if err != nil {
		log.Printf("Zygote could not be used to create child: %s", err.Error())
	}
	return sb, err
}

// responsible for acquiring and releasing Zygotes
func (tree *HugeTree) tryCreate(
	childSandboxPool sandbox.SandboxPool, createArgs CreateArgs) (sandbox.Sandbox, error) {

	zygoteP, zygoteC, err := tree.getZygotePair(createArgs.meta.Installs)
	if err != nil {
		return nil, err
	}

	defer zygoteC.release()
	if zygoteP != nil {
		defer zygoteP.release()
		if zygoteP.sb == nil {
			panic("a non-nil zygoteP must have a non-nil sandbox")
		}
	}
	log.Printf("Zygote Pair: %V, %V\n", zygoteP, zygoteC)

	return tree.tryCreateFromZygotes(childSandboxPool, createArgs, zygoteP, zygoteC)
}

// getZygotePair returns one or two Zygotes.  zygoteC (Child) will
// always be returned, but it may or may not have a sandbox.  zygoteP
// (Parent) may or may not be returned; if it is, it will definitely
// have a sandbox that could be used to fork a sandbox for zygoteC
//
// initialization will look something like this:
//
// root => ... => zygoteP => zygoteC => ... => handler
func (tree *HugeTree) getZygotePair(packages []string) (zygoteP *Zygote, zygoteC *Zygote, err error) {
	zygoteIDs := []int{}
	tree.root.FindEligibleZygotes(packages, &zygoteIDs)

	// zygoteC (child) is the one we'll use to create the handler
	// Sandbox, and zygoteP (parent) is the one we'll use to
	// create a Sandbox for zygoteC
	zygoteC = nil
	zygoteP = nil

	// prefer later in the last, because those Zygotes have more
	// of the packages that we need
	for i := len(zygoteIDs)-1; i >= 0; i-- {
		// try to get a Zygote at this level that already has a sandbox
		set := tree.zygoteSets[zygoteIDs[i]]
		set.mutex.Lock()
		if err := tree.initCodeDirIfNecessary(set, tree.nodes[zygoteIDs[i]]); err != nil {
			set.mutex.Unlock()
			return nil, nil, err
		}
		zygote := tree.zygoteSets[zygoteIDs[i]].reserve(true)
		set.mutex.Unlock()

		if zygote != nil {
			if i == len(zygoteIDs) - 1 {
				// yay, the deepest eligible Zygote
				// has a sandbox!
				zygoteC = zygote
			} else {
				// the zygote was not as deep as we
				// would have liked, so we'll use it
				// to create another zygote that is
				// one level deeper
				zygoteP = zygote
				set := tree.zygoteSets[zygoteIDs[i + 1]]
				set.mutex.Lock()
				zygoteC = set.reserve(false)
				set.mutex.Unlock()
			}

			return zygoteP, zygoteC, nil
		}
	}

	// could not find a Zygote with a sandbox at any level, so
	// accept one at the root that does not have a sandbox
	set := tree.zygoteSets[0]
	set.mutex.Lock()
	if err := tree.initCodeDirIfNecessary(set, tree.root); err != nil {
		set.mutex.Unlock()
		return nil, nil, err
	}
	zygoteC = set.reserve(false)
	set.mutex.Unlock()
	return zygoteP, zygoteC, nil
}

// responsible for pausing/unpausing sandboxes
func (tree *HugeTree) tryCreateFromZygotes(
	childSandboxPool sandbox.SandboxPool, createArgs CreateArgs,
	zygoteP *Zygote, zygoteC *Zygote) (sandbox.Sandbox, error) {

	// CASES
	// case 1: if zygoteC has a sandbox, use it
	// case 2: otherwise, if we have zygoteP, use zygoteP to create a sandbox for zygoteC, then use it
	// case 3: otherwise, create a sanbox for zygoteC (without any parent), then use it

	// if zygoteC has a sandbox, unpause it.  Otherwise, create
	// it.
	if zygoteC.sb != nil {
		if err := zygoteC.sb.Unpause(); err != nil {
			zygoteC.sb = nil
			return nil, err
		}
	} else {
		setC := zygoteC.containingSet
		scratchDir := tree.scratchDirs.Make("import-cache")

		// we have to create a zygoteC sandbox.  Do we have a
		// parent from which to create it, or do we need a
		// parentless sandbox?
		if zygoteP == nil {
			// we must create a parentless sandbox for zygoteC
			sb, err := tree.sbPool.Create(
				nil,   // no parent
				false, // not a leaf
				setC.codeDir,
				scratchDir,
				setC.meta,
				createArgs.rt_type)
			if err != nil {
				return nil, err
			}
			zygoteC.sb = sb
		} else {
			// we must create a sandbox for zygoteC from the sandbox in zygoteP
			if err := zygoteP.sb.Unpause(); err != nil {
				zygoteP.sb = nil
				return nil, err
			}
			defer zygoteP.sb.Pause()

			sb, err := tree.sbPool.Create(
				zygoteP.sb,   // this zygote has a parent zygote
				false,        // not a leaf
				setC.codeDir,
				scratchDir,
				setC.meta,
				createArgs.rt_type)
			if err != nil {
				return nil, err
			}
			zygoteC.sb = sb
		}
	}
	// if we get to here, we are guaranteed to have an unpaused sandbox in zygoteC
	defer zygoteC.sb.Pause()

	log.Printf("Creating CHILD from ZYGOTE!")
	return childSandboxPool.Create(
		zygoteC.sb, createArgs.isLeaf,
		createArgs.codeDir, createArgs.scratchDir,
		createArgs.meta, createArgs.rt_type)

}

func (tree *HugeTree) Cleanup() {
	for _, set := range tree.zygoteSets {
		set.mutex.Lock() // never unlock, because we should never use it again
		for _, zygote := range set.zygotes {
			if zygote.sb != nil {
				zygote.sb.Destroy("Zygote tree, final Cleanup()")
			}
		}
	}
}

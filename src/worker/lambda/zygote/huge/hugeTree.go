package huge

import (
	"log"
	"sync"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/lambda/packages"
	"github.com/open-lambda/open-lambda/ol/worker/sandbox"
)

type HugeTree struct {
	root *Node

	// the indexes between nodes and zygoteSets align
	nodes []*Node
	zygoteSets []*ZygoteSet
}

type Zygote struct {
	containingSet *ZygoteSet
	inUse         bool
	sb            sandbox.Sandbox
}

type ZygoteSet struct {
	mutex      sync.Mutex
	zygotes []*Zygote
}

func NewHugeTree(
	codeDirs *common.DirMaker,
	scratchDirs *common.DirMaker,
	sbPool sandbox.SandboxPool,
	pp *packages.PackagePuller,
) (*HugeTree, error) {
	nodes, err := LoadTreeFromConfig()
	if err != nil {
		return nil, err
	}

	zygoteSets := []*ZygoteSet{}
	for _ = range nodes {
		zygoteSets = append(zygoteSets, &ZygoteSet{})
	}

	return &HugeTree{
		nodes: nodes,
		root: nodes[0],
		zygoteSets: zygoteSets,
	}, nil
}

// reserve tries to give us a Zygote.  If sbMustExist is true, reserve
// will either return nil, or a Zygote with a sandbox.  If sbMustExist
// is false, we are guaranteed to return a Zygote, but it may or may
// not contain a Sandbox.
func (set *ZygoteSet) reserve(sbMustExist bool) *Zygote {
	set.mutex.Lock()
	defer set.mutex.Unlock()

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

	zygoteP, zygoteC := tree.getZygotePair(meta.Installs)
	defer zygoteC.release()
	if zygoteP != nil {
		defer zygoteP.release()
		if zygoteP.sb == nil {
			panic("a non-nil zygoteP must have a non-nil sandbox")
		}
	}
	log.Printf("Zygote Pair: %V, %V\n", zygoteP, zygoteC)

	// TODO case:
	// case 1: if zygoteC has a sandbox, use it
	// case 2: otherwise, if we have zygoteP, use zygoteP to create a sandbox for zygoteC, then use it
	// case 3: otherwise, create a sanbox for zygoteC (without any parent), then use it

	return nil, nil
}

// getZygotePair returns one or two Zygotes.  zygoteC (Child) will
// always be returned, but it may or may not have a sandbox.  zygoteP
// (Parent) may or may not be returned; if it is, it will definitely
// have a sandbox that could be used to fork a sandbox for zygoteC
//
// initialization will look something like this:
//
// root => ... => zygoteP => zygoteC => ... => handler
func (tree *HugeTree) getZygotePair(packages []string) (zygoteP *Zygote, zygoteC *Zygote) {
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
		zygote := tree.zygoteSets[zygoteIDs[i]].reserve(true)
		if zygote != nil {
			if i == len(zygoteIDs) - 1 {
				// yay, the deepest eligible Zygote has a sandbox!
				zygoteC = zygote
			} else {
				// the zygote was not as deep as we
				// would have liked, so we'll use it
				// to create another zygote that is
				// one level deeper
				zygoteP = zygote
				zygoteC = tree.zygoteSets[zygoteIDs[i + 1]].reserve(false)
			}
			break
		}
	}
	if zygoteC == nil {
		// could not find a Zygote with a sandbox at any level, so
		// accept one at the root that does not have a sandbox
		zygoteC = tree.zygoteSets[0].reserve(false)
	}

	return zygoteP, zygoteC
}

func (tree *HugeTree) Cleanup() {
	// TODO
}

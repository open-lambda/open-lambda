package zygote

import (
	"log"
	"math/rand"
	"runtime"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/lambda/packages"
	"github.com/open-lambda/open-lambda/ol/worker/sandbox"
)

// MultiTree is a ZygoteProvider that manages multiple ImportCache trees.
type MultiTree struct {
	trees []*ImportCache
}

// NewMultiTree creates a new MultiTree instance with the specified number of ImportCache trees.
func NewMultiTree(codeDirs *common.DirMaker, scratchDirs *common.DirMaker, sbPool sandbox.SandboxPool, pp *packages.PackagePuller) (*MultiTree, error) {
	var tree_count int
	switch cpus := runtime.NumCPU(); {
	case cpus < 3:
		tree_count = 6
	case cpus > 10:
		tree_count = 16
	default:
		tree_count = cpus * 2
	}
	log.Printf("Starting MultiTree ZygoteProvider with %d trees (tree count equals CPU count, with min of 3 and max of 10).", tree_count)

	trees := make([]*ImportCache, tree_count)
	for i := range trees {
		tree, err := NewImportCache(codeDirs, scratchDirs, sbPool, pp)
		if err != nil {
			for j := 0; j < i; j++ {
				trees[j].Cleanup()
			}
			return nil, err
		}
		trees[i] = tree
	}
	return &MultiTree{trees: trees}, nil
}

// Create creates a new sandbox using a randomly selected ImportCache tree.
func (mt *MultiTree) Create(childSandboxPool sandbox.SandboxPool, isLeaf bool, codeDir, scratchDir string, meta *sandbox.SandboxMeta, rt_type common.RuntimeType) (sandbox.Sandbox, error) {
	idx := rand.Intn(len(mt.trees))
	return mt.trees[idx].Create(childSandboxPool, isLeaf, codeDir, scratchDir, meta, rt_type)
}

// Cleanup performs cleanup operations for all ImportCache trees in the MultiTree.
func (mt *MultiTree) Cleanup() {
	for _, tree := range mt.trees {
		tree.Cleanup()
	}
}

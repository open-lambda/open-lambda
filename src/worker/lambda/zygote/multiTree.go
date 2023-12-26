package zygote

import (
	"log"
	"math/rand"
	"runtime"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/lambda/packages"
	"github.com/open-lambda/open-lambda/ol/worker/sandbox"
)

type MultiTree struct {
	trees []*ImportCache
}

func (mt *MultiTree) Warmup() error {
	panic("multi-tree warmup not implemented!")
}

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
			for j := 0; j < i; j ++ {
				trees[j].Cleanup()
			}
			return nil, err
		}
		trees[i] = tree
	}
	return &MultiTree{trees: trees}, nil
}

func (mt *MultiTree) Create(childSandboxPool sandbox.SandboxPool, isLeaf bool, codeDir, scratchDir string, meta *sandbox.SandboxMeta, rt_type common.RuntimeType) (sandbox.Sandbox, error) {
	idx := rand.Intn(len(mt.trees))
	return mt.trees[idx].Create(childSandboxPool, isLeaf, codeDir, scratchDir, meta, rt_type)
}

func (mt *MultiTree) Cleanup() {
	for _, tree := range mt.trees {
		tree.Cleanup()
	}
}

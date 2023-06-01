package zygote

import (
	"log"
	"fmt"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/lambda/packages"
	"github.com/open-lambda/open-lambda/ol/worker/sandbox"
)

func NewZygoteProvider(codeDirs *common.DirMaker, scratchDirs *common.DirMaker, sbPool sandbox.SandboxPool, pp *packages.PackagePuller) (ZygoteProvider, error) {
	switch common.Conf.Features.Import_cache {
	case "tree":
		return NewImportCache(codeDirs, scratchDirs, sbPool, pp)
	case "multitree":
		log.Printf("'multitree' zygote provider is very experimental")
		return NewMultiTree(codeDirs, scratchDirs, sbPool, pp)
	default:
		return nil, fmt.Errorf("provider %s not implemented")
	}
}

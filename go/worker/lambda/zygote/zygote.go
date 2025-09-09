package zygote

import (
	"fmt"
	"log/slog"

	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/lambda/packages"
	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// NewZygoteProvider creates a new ZygoteProvider based on the specified import cache implementation.
func NewZygoteProvider(codeDirs *common.DirMaker, scratchDirs *common.DirMaker, sbPool sandbox.SandboxPool, pp *packages.PackagePuller) (ZygoteProvider, error) {
	switch impl := common.Conf.Features.Import_cache; impl {
	case "tree":
		return NewImportCache(codeDirs, scratchDirs, sbPool, pp)
	case "multitree":
		slog.Info(fmt.Sprintf("ZygoteProvider %s is very experimental.", impl))
		return NewMultiTree(codeDirs, scratchDirs, sbPool, pp)
	default:
		return nil, fmt.Errorf("ZygoteProvider '%s' is not implemented", impl)
	}
}

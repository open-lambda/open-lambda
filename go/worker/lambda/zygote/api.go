package zygote

import (
	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/sandbox"
)

type ZygoteProvider interface {
	Create(childSandboxPool sandbox.SandboxPool, isLeaf bool,
		codeDir, scratchDir string, meta *sandbox.SandboxMeta,
		rt_type common.RuntimeType) (sandbox.Sandbox, error)
	Cleanup()
}

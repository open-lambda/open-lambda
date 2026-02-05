package zygote

import (
	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

type ZygoteProvider interface {
	Create(childSandboxPool sandbox.SandboxPool, isLeaf bool,
		codeDir, scratchDir string, meta *sandbox.SandboxMeta) (sandbox.Sandbox, error)
	Cleanup()
}

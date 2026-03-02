package sandboxset

import (
	"fmt"
	"sync"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// sandboxWrapper pairs a sandbox with an in-use flag.
type sandboxWrapper struct {
	sb    sandbox.Sandbox
	inUse bool
}

// sandboxSetImpl is the private concrete type returned by New.
// All mutable state is guarded by mu.
type sandboxSetImpl struct {
	mu     sync.Mutex
	pool   []*sandboxWrapper
	cfg    *Config
	closed bool
}

func newSandboxSet(cfg *Config) (*sandboxSetImpl, error) {
	if cfg == nil {
		return nil, fmt.Errorf("sandboxset: Config must not be nil")
	}
	if cfg.Pool == nil {
		return nil, fmt.Errorf("sandboxset: Config.Pool must not be nil")
	}
	if cfg.CodeDir == "" {
		return nil, fmt.Errorf("sandboxset: Config.CodeDir must not be empty")
	}
	if cfg.ScratchDirs == nil {
		return nil, fmt.Errorf("sandboxset: Config.ScratchDirs must not be nil")
	}
	return &sandboxSetImpl{cfg: cfg}, nil
}

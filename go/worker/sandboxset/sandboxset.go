package sandboxset

import (
	"fmt"
	"sync"
	"time"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// sandboxWrapper pairs a sandbox with an in-use flag.
// Never exported; internal pool bookkeeping only.
type sandboxWrapper struct {
	sb    sandbox.Sandbox
	inUse bool
}

// getOptions holds resolved settings for one GetSandbox call.
type getOptions struct {
	timeout time.Duration
}

// releaseOptions holds resolved settings for one ReleaseSandbox call.
type releaseOptions struct {
	skipPause bool
}

// sandboxSetImpl is the private concrete type returned by New.
// All mutable state is guarded by mu.
type sandboxSetImpl struct {
	mu      sync.RWMutex
	pool    []*sandboxWrapper
	cfg     *Config
	metrics Metrics
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
	return &sandboxSetImpl{cfg: cfg}, nil
}

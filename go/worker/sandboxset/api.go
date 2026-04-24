// Package sandboxset provides a thread-safe pool of sandboxes for a single Lambda function. Callers ask for a sandbox and don't worry about whether it is freshly created or recycled from a previous request.
//
// Usage:
//
//	set := sandboxset.New(&sandboxset.Config{
//	    Pool:        myPool,
//	    CodeDir:     "/path/to/lambda",
//	    ScratchDirs: myScratchDirs,
//	})
//
//	ref, err := set.GetOrCreateUnpaused()
//	// ... use ref.Sandbox() to handle request ...
//	if broken {
//	    ref.MarkDead()
//	}
//	ref.Put()
package sandboxset

import (
	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// SandboxSet manages a pool of sandboxes for one Lambda function.
// All methods are safe to call from multiple goroutines.
type SandboxSet interface {
	// GetOrCreateUnpaused returns an unpaused sandbox ready to handle a
	// request, wrapped in a SandboxRef.
	GetOrCreateUnpaused() (*SandboxRef, error)

	// Close destroys all sandboxes in the pool and marks the set as closed.
	Close() error
}

// Config holds the parameters needed to create a SandboxSet.
type Config struct {
	// Pool creates and destroys the underlying sandboxes.
	Pool sandbox.SandboxPool

	// Parent is an optional SandboxSet to fork from. When nil, new
	// sandboxes are created from scratch. Not all SandboxPool
	// implementations support forking. The parent must outlive this child.
	Parent SandboxSet

	// IsLeaf marks sandboxes as non-forkable, meaning they will
	// not be used as parents for future forks.
	IsLeaf bool

	// CodeDir is the directory containing the Lambda handler code.
	CodeDir string

	// Meta holds runtime configuration (memory limits, packages,
	// imports, etc.). Nil means the pool fills in defaults.
	Meta *sandbox.SandboxMeta

	// ScratchDirs creates a unique writable directory for each
	// new sandbox. The set calls ScratchDirs.Make internally
	// so that GetOrCreateUnpaused can remain argument-free.
	ScratchDirs *common.DirMaker
}

// New creates a SandboxSet from cfg.
// Panics if Pool, CodeDir, or ScratchDirs are missing.
func New(cfg *Config) SandboxSet {
	return newSandboxSet(cfg)
}

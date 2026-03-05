// Package sandboxset provides a thread-safe pool of sandboxes for a single
// Lambda function.
//
// A SandboxSet replaces per-instance goroutines with a simple pool.
// Callers just ask for a sandbox and don't worry about whether it is
// freshly created or recycled from a previous request.
//
// Sandbox lifecycle inside a SandboxSet:
//
//	[created]
//	    |
//	    v
//	[paused]  <---+
//	    |         |
//	    v         |
//	[in-use]  ----+  (Put)
//	    |
//	    v
//	[destroyed]     (Destroy / Close / error)
//
// Usage:
//
//	set, err := sandboxset.New(&sandboxset.Config{
//	    Pool:        myPool,
//	    CodeDir:     "/path/to/lambda",
//	    ScratchDirs: myScratchDirs,
//	})
//
//	sb, err := set.Get()
//	// ... handle request ...
//	set.Put(sb)
package sandboxset

import (
	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

/*
A SandboxSet manages a pool of sandboxes for one Lambda function.
All methods are safe to call from multiple goroutines.

The design mirrors the C process API: Get (create), Put (exit),
Destroy (kill), Close (cleanup). There are no warm-up, shrink, or
stats methods yet — those can be added in later PRs without
changing the core interface.
*/
type SandboxSet interface {
	// Return an unpaused sandbox ready to handle a request.
	//
	// If the pool has an idle sandbox, it is unpaused and returned.
	// If Unpause fails (e.g., the SOCK container died while paused),
	// that sandbox is destroyed and Get tries the next idle one or
	// creates a fresh sandbox.
	//
	// A fresh scratch directory is created for each new sandbox
	// via Config.ScratchDirs. Reused sandboxes keep their
	// existing scratch directory from when they were first created.
	Get() (sandbox.Sandbox, error)

	// Return a sandbox to the pool after a successful request.
	//
	// The sandbox is paused and becomes available for the next Get.
	// If Pause fails (e.g., the container died during the request),
	// the sandbox is destroyed automatically — a bad sandbox never
	// re-enters the pool.
	//
	// Passing a sandbox that is not in the pool returns an error
	// but is otherwise harmless. If the set has been closed, Put
	// returns an error immediately — the sandbox was already
	// destroyed by Close.
	Put(sb sandbox.Sandbox) error

	// Permanently remove a sandbox from the pool and destroy it.
	//
	// Use this when a request produced an unrecoverable error and
	// the sandbox should not be reused. "reason" is a
	// human-readable explanation that shows up in later error
	// messages (same convention as sandbox.Sandbox.Destroy).
	//
	// If the sandbox is not in the pool it is still destroyed —
	// resources are always freed. The returned error is
	// informational only.
	Destroy(sb sandbox.Sandbox, reason string) error

	// Destroy all sandboxes in the pool and mark the set as closed.
	//
	// Callers who still hold sandbox references from a previous Get
	// will find them already dead, which is safe: per the Sandbox
	// contract, methods on a destroyed sandbox are harmless no-ops
	// that return errors.
	//
	// Calling Close a second time returns an error.
	Close() error
}

// Config holds the parameters needed to create a SandboxSet.
type Config struct {
	// Pool creates and destroys the underlying sandboxes.
	Pool sandbox.SandboxPool

	// Parent sandbox to fork from (may be nil). When nil, new
	// sandboxes are created from scratch. Not all SandboxPool
	// implementations support forking.
	Parent sandbox.Sandbox

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
	// so that Get can remain argument-free.
	ScratchDirs *common.DirMaker
}

// New creates a SandboxSet from cfg. Returns an error if any of
// Pool, CodeDir, or ScratchDirs are missing.
func New(cfg *Config) (SandboxSet, error) {
	return newSandboxSet(cfg)
}

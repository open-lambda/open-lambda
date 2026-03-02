// Package sandboxset provides a thread-safe pool of sandboxes for a single
// Lambda function.
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

// SandboxSet is a thread-safe pool of sandboxes for a single Lambda function.
// All methods are safe for concurrent use.
type SandboxSet interface {
	// Get borrows a sandbox, creating one if none are idle.
	// The returned sandbox is unpaused and ready to handle a request.
	// Caller MUST call Put or Destroy when done.
	Get() (sandbox.Sandbox, error)

	// Put returns a sandbox to the pool after successful use.
	// The sandbox is paused and made available for future Get calls.
	// If pausing fails, the sandbox is destroyed automatically.
	Put(sb sandbox.Sandbox) error

	// Destroy permanently removes a sandbox from the pool and kills it.
	// Use when a sandbox is in an unrecoverable state.
	Destroy(sb sandbox.Sandbox, reason string) error

	// Close destroys all sandboxes in the pool.
	// After Close returns, Get/Put/Destroy return errors.
	Close() error
}

// Config holds creation parameters for a SandboxSet.
// Pool, CodeDir, and ScratchDirs are required.
type Config struct {
	// Pool creates new sandboxes. Required.
	Pool sandbox.SandboxPool

	// Parent is the sandbox to fork from. Nil means create from scratch.
	Parent sandbox.Sandbox

	// IsLeaf marks created sandboxes as non-forkable.
	IsLeaf bool

	// CodeDir is the Lambda function code directory. Required.
	CodeDir string

	// Meta holds runtime configuration. Nil means pool defaults.
	Meta *sandbox.SandboxMeta

	// ScratchDirs creates per-sandbox writable directories. Required.
	ScratchDirs *common.DirMaker
}

// New creates a SandboxSet. Returns an error if cfg is invalid.
func New(cfg *Config) (SandboxSet, error) {
	return newSandboxSet(cfg)
}

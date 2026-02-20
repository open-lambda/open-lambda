// Package sandboxset provides a thread-safe pool of sandboxes for a single
// Lambda function.
//
// SandboxSet replaces per-instance goroutines with a mutex-protected slice.
// The pool costs ~500 bytes regardless of how many sandboxes it holds.
//
// Sandbox lifecycle inside a SandboxSet:
//
//	created ──► paused (available) ──► in-use (unpaused) ──► paused (available)
//	                  │                       │
//	                  └───────── destroyed ◄──┘  (on error or Shrink)
//
// Usage:
//
//	cfg := &sandboxset.Config{
//	    Pool:    myPool,
//	    CodeDir: "/path/to/lambda",
//	    Meta:    &sandbox.SandboxMeta{Runtime: common.RT_PYTHON},
//	}
//	set, err := sandboxset.New(cfg)
//
//	sb, err := set.GetSandbox()
//	// ... handle request ...
//	set.ReleaseSandbox(sb)
package sandboxset

import (
	"time"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// SandboxSet is a thread-safe pool of sandboxes for a single Lambda function.
// All methods are safe to call from multiple goroutines concurrently.
type SandboxSet interface {
	// GetSandbox borrows an available sandbox, creating one if needed.
	//
	// Returns an unpaused sandbox ready to handle a request.
	// Caller MUST release it via ReleaseSandbox or DestroyAndRemove.
	// Blocks up to DefaultTimeout (or WithTimeout) when at MaxSize capacity.
	GetSandbox(opts ...GetOption) (sandbox.Sandbox, error)

	// ReleaseSandbox returns a sandbox to the pool after a successful request.
	// The sandbox is paused and made available for the next caller.
	// On Pause failure the sandbox is destroyed automatically.
	ReleaseSandbox(sb sandbox.Sandbox, opts ...ReleaseOption) error

	// DestroyAndRemove permanently removes a sandbox from the pool.
	// Use when a sandbox has produced an unrecoverable error.
	DestroyAndRemove(sb sandbox.Sandbox, reason string) error

	// Warm pre-creates sandboxes until at least target are paused and ready.
	// No-op when the pool already has target or more sandboxes.
	Warm(target int) error

	// Shrink destroys idle sandboxes until at most target remain.
	// Stops early if all remaining sandboxes are in use.
	Shrink(target int) error

	// Stats returns a snapshot of pool counters.
	// Keys: "available", "in_use", "total".
	Stats() map[string]int

	// Metrics returns a copy of cumulative performance counters.
	Metrics() *Metrics
}

// Config holds creation parameters for a SandboxSet.
// Pool and CodeDir are required; all other fields have sensible zero-value defaults.
type Config struct {
	// Pool creates new sandboxes. Required.
	Pool sandbox.SandboxPool

	// Parent is the sandbox to fork from. Nil means create from scratch.
	Parent sandbox.Sandbox

	// IsLeaf specifies whether created sandboxes are leaves (not forkable).
	IsLeaf bool

	// CodeDir is the directory containing the Lambda function code. Required.
	CodeDir string

	// ScratchDir is the per-invocation writable directory for each sandbox.
	ScratchDir string

	// Meta holds runtime configuration (memory limits, packages, imports).
	Meta *sandbox.SandboxMeta

	// MaxSize caps the total number of sandboxes. Zero means no limit.
	MaxSize int

	// DefaultTimeout is how long GetSandbox blocks at MaxSize capacity.
	// Zero means fail immediately when all sandboxes are in use.
	DefaultTimeout time.Duration
}

// GetOption adjusts a single GetSandbox call.
// Construct with WithTimeout.
type GetOption func(*getOptions)

// ReleaseOption adjusts a single ReleaseSandbox call.
// Construct with WithoutPause.
type ReleaseOption func(*releaseOptions)

// Metrics holds cumulative counters since pool creation.
// All fields are monotonically increasing.
type Metrics struct {
	Gets     int64 // total GetSandbox calls
	Hits     int64 // Gets served from an already-available sandbox
	Misses   int64 // Gets that required creating a new sandbox
	Releases int64 // successful ReleaseSandbox calls
	Destroys int64 // DestroyAndRemove calls
	Timeouts int64 // Gets that failed due to deadline exceeded
}

// HitRate returns the fraction of Gets served from an available sandbox (0.0–1.0).
func (m *Metrics) HitRate() float64 {
	if m.Gets == 0 {
		return 0.0
	}
	return float64(m.Hits) / float64(m.Gets)
}

// New creates a SandboxSet from cfg. Returns an error if cfg is invalid.
func New(cfg *Config) (SandboxSet, error) {
	return newSandboxSet(cfg)
}

// WithTimeout overrides the DefaultTimeout for a single GetSandbox call.
func WithTimeout(d time.Duration) GetOption {
	return func(o *getOptions) { o.timeout = d }
}

// WithoutPause skips the Pause call when returning a sandbox to the pool.
// Use when the sandbox has already been paused by the caller.
func WithoutPause() ReleaseOption {
	return func(o *releaseOptions) { o.skipPause = true }
}

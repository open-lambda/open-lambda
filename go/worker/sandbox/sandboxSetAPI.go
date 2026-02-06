package sandbox

import (
	"time"
)

// SandboxSetProvider manages pools of sandboxes for lambda functions.
// It eliminates the need for per-instance goroutines by providing a
// simple, thread-safe pool abstraction.
//
// Design Philosophy:
// - Thread-safe: All operations protected by mutex
// - Elastic: Dynamically scales up to maxSandboxes
// - Resilient: Handles broken sandboxes gracefully
// - Observable: Provides statistics for monitoring
type SandboxSetProvider interface {
	// GetSandbox returns an available sandbox or creates a new one.
	// Returns error if all sandboxes are busy and max capacity reached.
	//
	// The returned sandbox is marked "in-use" and will not be given
	// to other callers until ReleaseSandbox() is called.
	//
	// Thread-safe: Multiple goroutines can call simultaneously.
	GetSandbox() (Sandbox, error)

	// ReleaseSandbox marks a sandbox as available for reuse.
	// The sandbox remains in the pool for future GetSandbox() calls.
	//
	// If the sandbox is detected to be unhealthy during release,
	// it will be automatically destroyed and removed from the pool.
	//
	// Returns error if:
	// - sb is nil
	// - sb was not borrowed from this set
	// - sb is already released (double-release)
	//
	// Thread-safe: Multiple goroutines can call simultaneously.
	ReleaseSandbox(sb Sandbox) error

	// DestroyAndRemove destroys a sandbox and removes it from the pool.
	// Use this when a sandbox is detected to be dead/corrupted and
	// should not be reused.
	//
	// This is useful when the caller knows a sandbox is broken
	// (e.g., after a failed function execution or OOM).
	//
	// Returns error if:
	// - sb is nil
	// - sb not found in this set
	//
	// Thread-safe: Safe to call from error handlers.
	DestroyAndRemove(sb Sandbox) error

	// Destroy cleans up all sandboxes in the set.
	// Called during worker shutdown.
	//
	// After calling Destroy(), the SandboxSet should not be used.
	Destroy()

	// Stats returns current pool statistics for monitoring/debugging.
	// Returns a map with keys:
	// - "total": current number of sandboxes in pool
	// - "in_use": sandboxes currently borrowed
	// - "idle": sandboxes available for reuse
	// - "max": maximum capacity
	// - "created": lifetime count of sandboxes created
	// - "borrowed": lifetime count of GetSandbox() calls
	// - "released": lifetime count of ReleaseSandbox() calls
	// - "removed": lifetime count of destroyed sandboxes
	// - "health_checks": lifetime count of health checks performed
	// - "health_failures": lifetime count of failed health checks
	//
	// Thread-safe: Safe to call from monitoring goroutines.
	Stats() map[string]int

	// Size returns the current number of sandboxes in the pool.
	// Equivalent to Stats()["total"] but more efficient.
	Size() int

	// InUse returns the number of sandboxes currently borrowed.
	// Equivalent to Stats()["in_use"] but more efficient.
	InUse() int
}

// SandboxSetConfig configures a SandboxSet's behavior.
type SandboxSetConfig struct {
	// MaxSandboxes is the maximum number of sandboxes in the pool.
	// If <= 0, defaults to 10.
	MaxSandboxes int

	// HealthCheckOnRelease: if true, check sandbox health when released.
	// Unhealthy sandboxes are automatically destroyed and removed.
	// Default: false (for backward compatibility)
	HealthCheckOnRelease bool

	// HealthCheckTimeout: maximum time to wait for health check.
	// Only used if HealthCheckOnRelease is true.
	// Default: 1 second
	HealthCheckTimeout time.Duration
}

// DefaultSandboxSetConfig returns sensible defaults.
func DefaultSandboxSetConfig() *SandboxSetConfig {
	return &SandboxSetConfig{
		MaxSandboxes:         10,
		HealthCheckOnRelease: false,
		HealthCheckTimeout:   1 * time.Second,
	}
}

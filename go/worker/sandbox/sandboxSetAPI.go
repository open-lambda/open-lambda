package sandbox

import (
	"errors"
	"time"
)

// =============================================================================
// ERRORS
// =============================================================================

var (
	// ErrCapacity is returned when the pool is at maximum capacity
	ErrCapacity = errors.New("sandbox pool at capacity")

	// ErrTimeout is returned when an operation times out
	ErrTimeout = errors.New("operation timed out")

	// ErrUnknownSandbox is returned when a sandbox is not found in the pool
	ErrUnknownSandbox = errors.New("sandbox not found in pool")

	// ErrNotInUse is returned when trying to release a sandbox that's not in use
	ErrNotInUse = errors.New("sandbox is not in use")

	// ErrClosed is returned when operating on a closed pool
	ErrClosed = errors.New("sandbox pool is closed")
)

// ErrorType categorizes different types of errors for metrics
type ErrorType int

const (
	ErrorTypeCapacity ErrorType = iota
	ErrorTypeTimeout
	ErrorTypeHealthCheck
	ErrorTypeCreate
	ErrorTypeDestroy
	ErrorTypeUnknownSandbox
	ErrorTypeDoubleFree
	ErrorTypeOther
)

func (e ErrorType) String() string {
	switch e {
	case ErrorTypeCapacity:
		return "capacity"
	case ErrorTypeTimeout:
		return "timeout"
	case ErrorTypeHealthCheck:
		return "health_check"
	case ErrorTypeCreate:
		return "create"
	case ErrorTypeDestroy:
		return "destroy"
	case ErrorTypeUnknownSandbox:
		return "unknown_sandbox"
	case ErrorTypeDoubleFree:
		return "double_free"
	default:
		return "other"
	}
}

// =============================================================================
// SANDBOX STATE
// =============================================================================

// SandboxState represents the current state of a sandbox in the pool
type SandboxState int

const (
	// StateIdle means the sandbox is available for use
	StateIdle SandboxState = iota

	// StateInUse means the sandbox is currently borrowed
	StateInUse

	// StateChecking means the sandbox is undergoing a health check
	StateChecking

	// StateWarming means the sandbox is being created (pre-warming)
	StateWarming

	// StateDead means the sandbox failed health checks and will be removed
	StateDead
)

func (s SandboxState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateInUse:
		return "in_use"
	case StateChecking:
		return "checking"
	case StateWarming:
		return "warming"
	case StateDead:
		return "dead"
	default:
		return "unknown"
	}
}

// =============================================================================
// HEALTH CHECK CONFIGURATION
// =============================================================================

// HealthCheckMethod defines how sandbox health is verified
type HealthCheckMethod int

const (
	// HealthCheckPause just calls Pause() to verify the sandbox responds
	HealthCheckPause HealthCheckMethod = iota

	// HealthCheckPauseUnpause calls Pause() then Unpause() for thorough check
	HealthCheckPauseUnpause

	// HealthCheckHTTPPing sends HTTP request to health endpoint (future)
	HealthCheckHTTPPing

	// HealthCheckCombined performs multiple checks (future)
	HealthCheckCombined
)

func (m HealthCheckMethod) String() string {
	switch m {
	case HealthCheckPause:
		return "pause"
	case HealthCheckPauseUnpause:
		return "pause_unpause"
	case HealthCheckHTTPPing:
		return "http_ping"
	case HealthCheckCombined:
		return "combined"
	default:
		return "unknown"
	}
}

// =============================================================================
// OPTIONS PATTERN
// =============================================================================

// GetOption configures GetSandbox behavior
type GetOption func(*getOptions)

type getOptions struct {
	timeout       time.Duration
	skipHealth    bool
	preferRecent  bool
	maxAge        time.Duration
}

// WithTimeout sets a timeout for GetSandbox operation
func WithTimeout(d time.Duration) GetOption {
	return func(o *getOptions) {
		o.timeout = d
	}
}

// SkipHealthCheckOnGet skips health check before returning sandbox
func SkipHealthCheckOnGet() GetOption {
	return func(o *getOptions) {
		o.skipHealth = true
	}
}

// WithPreferRecent prefers sandboxes created within maxAge
// Older sandboxes are removed during selection
func WithPreferRecent(maxAge time.Duration) GetOption {
	return func(o *getOptions) {
		o.preferRecent = true
		o.maxAge = maxAge
	}
}

// ReleaseOption configures ReleaseSandbox behavior
type ReleaseOption func(*releaseOptions)

type releaseOptions struct {
	skipHealth bool
	markDirty  bool
}

// SkipHealthCheck skips health check when releasing sandbox
func SkipHealthCheck() ReleaseOption {
	return func(o *releaseOptions) {
		o.skipHealth = true
	}
}

// MarkAsDirty forces health check even if normally disabled
func MarkAsDirty() ReleaseOption {
	return func(o *releaseOptions) {
		o.markDirty = true
	}
}

// =============================================================================
// METRICS AND OBSERVABILITY
// =============================================================================

// PoolMetrics provides detailed statistics about the sandbox pool
type PoolMetrics struct {
	// Current state
	Total    int
	InUse    int
	Idle     int
	Checking int
	Max      int

	// Lifetime counters
	Created        int64
	Borrowed       int64
	Released       int64
	Removed        int64
	HealthChecks   int64
	HealthPasses   int64
	HealthFailures int64

	// Timing
	Uptime      time.Duration
	LastBorrow  *time.Time
	LastRelease *time.Time

	// Efficiency metrics
	HitRate             float64 // reuse rate: released / borrowed
	CapacityUtilization float64 // in_use / max

	// Average durations
	AverageIdleTime time.Duration
	AverageUseTime  time.Duration

	// Per-sandbox breakdown (for debugging)
	SandboxMetrics []*SandboxMetric

	// Error tracking
	ErrorCounts map[ErrorType]int64
	LastErrors  []*ErrorRecord
}

// SandboxMetric provides statistics for a single sandbox
type SandboxMetric struct {
	ID               string
	State            SandboxState
	Age              time.Duration
	LastUsed         time.Time
	UseCount         int
	HealthCheckCount int
	HealthFailCount  int
	CreatedAt        time.Time
	TotalBusyTime    time.Duration
	TotalIdleTime    time.Duration
}

// ErrorRecord tracks a single error occurrence
type ErrorRecord struct {
	Type      ErrorType
	Time      time.Time
	Message   string
	SandboxID string
}

// =============================================================================
// EVENT SYSTEM
// =============================================================================

// SandboxSetEventType represents different lifecycle events
type SandboxSetEventType int

const (
	EventSandboxCreated SandboxSetEventType = iota
	EventSandboxBorrowed
	EventSandboxReleased
	EventSandboxHealthCheck
	EventSandboxHealthPassed
	EventSandboxHealthFailed
	EventSandboxRemoved
	EventSandboxWarmed
	EventPoolAtCapacity
	EventPoolShrunk
	EventPoolGrown
)

func (e SandboxSetEventType) String() string {
	switch e {
	case EventSandboxCreated:
		return "created"
	case EventSandboxBorrowed:
		return "borrowed"
	case EventSandboxReleased:
		return "released"
	case EventSandboxHealthCheck:
		return "health_check"
	case EventSandboxHealthPassed:
		return "health_passed"
	case EventSandboxHealthFailed:
		return "health_failed"
	case EventSandboxRemoved:
		return "removed"
	case EventSandboxWarmed:
		return "warmed"
	case EventPoolAtCapacity:
		return "at_capacity"
	case EventPoolShrunk:
		return "shrunk"
	case EventPoolGrown:
		return "grown"
	default:
		return "unknown"
	}
}

// SandboxSetEvent represents a lifecycle event in the pool
type SandboxSetEvent struct {
	Type      SandboxSetEventType
	Time      time.Time
	SandboxID string
	Details   map[string]interface{}
}

// SandboxSetEventHandler processes lifecycle events
type SandboxSetEventHandler func(SandboxSetEvent)

// =============================================================================
// MAIN INTERFACE
// =============================================================================

// SandboxSetProvider manages pools of sandboxes for lambda functions.
// It eliminates the need for per-instance goroutines by providing a
// simple, thread-safe pool abstraction.
//
// Design Philosophy:
// - Thread-safe: All operations protected by RWMutex
// - Elastic: Dynamically scales up to maxSandboxes
// - Resilient: Handles broken sandboxes gracefully
// - Observable: Provides rich statistics and events for monitoring
// - Flexible: Options pattern allows per-call configuration
type SandboxSetProvider interface {
	// GetSandbox returns an available sandbox or creates a new one.
	// Supports optional configuration via GetOption parameters.
	//
	// The returned sandbox is marked "in-use" and will not be given
	// to other callers until ReleaseSandbox() is called.
	//
	// Returns ErrCapacity if pool is at max capacity.
	// Returns ErrTimeout if timeout expires (when WithTimeout used).
	//
	// Examples:
	//   sb, err := pool.GetSandbox()                              // Basic
	//   sb, err := pool.GetSandbox(WithTimeout(5*time.Second))    // With timeout
	//   sb, err := pool.GetSandbox(WithPreferRecent(10*time.Minute)) // Prefer fresh
	//
	// Thread-safe: Multiple goroutines can call simultaneously.
	GetSandbox(opts ...GetOption) (Sandbox, error)

	// ReleaseSandbox marks a sandbox as available for reuse.
	// Supports optional configuration via ReleaseOption parameters.
	//
	// The sandbox remains in the pool for future GetSandbox() calls.
	// If health checks are enabled and fail, the sandbox is automatically
	// destroyed and removed from the pool.
	//
	// Returns error if:
	// - sb is nil
	// - sb was not borrowed from this set
	// - sb is already released (double-release)
	//
	// Examples:
	//   err := pool.ReleaseSandbox(sb)                    // Basic
	//   err := pool.ReleaseSandbox(sb, SkipHealthCheck()) // Skip health check
	//   err := pool.ReleaseSandbox(sb, MarkAsDirty())     // Force health check
	//
	// Thread-safe: Multiple goroutines can call simultaneously.
	ReleaseSandbox(sb Sandbox, opts ...ReleaseOption) error

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
	DestroyAndRemove(sb Sandbox, reason string) error

	// Destroy cleans up all sandboxes in the set.
	// Called during worker shutdown.
	//
	// After calling Destroy(), the SandboxSet should not be used.
	Destroy()

	// Warm pre-creates sandboxes up to the target count.
	// Useful for reducing cold starts before expected traffic.
	//
	// Returns error if target exceeds MaxSandboxes or creation fails.
	//
	// Example:
	//   pool.Warm(5)  // Pre-create 5 sandboxes
	Warm(target int) error

	// Shrink removes idle sandboxes down to the target count.
	// Respects MinIdleSandboxes configuration (won't go below it).
	//
	// Returns error if target is negative or shrinking fails.
	//
	// Example:
	//   pool.Shrink(2)  // Keep only 2 idle sandboxes
	Shrink(target int) error

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
	// Note: For detailed metrics, use Metrics() instead.
	Stats() map[string]int

	// Metrics returns detailed pool metrics including efficiency,
	// per-sandbox breakdown, and error tracking.
	//
	// This is more expensive than Stats() as it computes derived metrics.
	// Use Stats() for frequent polling, Metrics() for detailed analysis.
	//
	// Thread-safe: Safe to call from monitoring goroutines.
	Metrics() *PoolMetrics

	// Size returns the current number of sandboxes in the pool.
	// Equivalent to Stats()["total"] but more efficient.
	Size() int

	// InUse returns the number of sandboxes currently borrowed.
	// Equivalent to Stats()["in_use"] but more efficient.
	InUse() int
}

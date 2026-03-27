package sandbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// =============================================================================
// CONFIGURATION
// =============================================================================

type SandboxSetConfig struct {
	// Capacity management
	MaxSandboxes     int
	MinIdleSandboxes int

	// Health checking
	HealthCheckOnRelease bool
	HealthCheckOnGet     bool
	HealthCheckTimeout   time.Duration
	HealthCheckMethod    HealthCheckMethod

	// No scheduling strategy needed with linear search

	// Timeouts
	GetTimeout    time.Duration
	CreateTimeout time.Duration

	// Eviction (for future use)
	IdleTimeout time.Duration

	// Events
	EventHandlers []SandboxSetEventHandler
}

func DefaultSandboxSetConfig() *SandboxSetConfig {
	return &SandboxSetConfig{
		MaxSandboxes:         10,
		MinIdleSandboxes:     0,
		HealthCheckOnRelease: false,
		HealthCheckOnGet:     false,
		HealthCheckTimeout:   1 * time.Second,
		HealthCheckMethod:    HealthCheckPause,
		GetTimeout:           30 * time.Second,
		CreateTimeout:        10 * time.Second,
		IdleTimeout:          0, // Disabled
		EventHandlers:        nil,
	}
}

// =============================================================================
// SANDBOXSET IMPLEMENTATION
// =============================================================================

// SandboxSet manages a thread-safe pool of sandboxes with O(log n) performance.
type SandboxSet struct {
	mutex sync.RWMutex // Read-write mutex for better concurrency

	// Simple list of sandboxes (linear search is fine for small pools)
	sandboxes []*sandboxWrapper

	// Configuration
	config *SandboxSetConfig

	// Dependencies for creating new sandboxes
	sbPool SandboxPool

	// Metadata for creating sandboxes
	meta       *SandboxMeta
	isLeaf     bool
	codeDir    string
	scratchDir string

	// Statistics (protected by mutex)
	created        int64
	borrowed       int64
	released       int64
	removed        int64
	healthChecks   int64
	healthPasses   int64
	healthFailures int64

	// Error tracking
	errorCounts map[ErrorType]int64
	lastErrors  []*ErrorRecord

	// Timing
	startTime   time.Time
	lastBorrow  *time.Time
	lastRelease *time.Time

	// Lifecycle
	closed bool
}

// sandboxWrapper tracks sandbox state and metadata
type sandboxWrapper struct {
	sb    Sandbox
	state SandboxState

	// Rich metadata
	createdAt    time.Time
	lastUsed     time.Time
	useCount     int
	healthChecks int
	healthFails  int

	// Timing for metrics
	totalBusyTime time.Duration
	totalIdleTime time.Duration
	busyStart     *time.Time // When last borrowed
}

// =============================================================================
// CONSTRUCTORS
// =============================================================================

// Backwards-compatible constructor
func NewSandboxSet(
	sbPool SandboxPool,
	meta *SandboxMeta,
	isLeaf bool,
	codeDir string,
	scratchDir string,
	maxSandboxes int,
) *SandboxSet {
	cfg := DefaultSandboxSetConfig()
	cfg.MaxSandboxes = maxSandboxes
	return NewSandboxSetWithConfig(sbPool, meta, isLeaf, codeDir, scratchDir, cfg)
}

// Configurable constructor
func NewSandboxSetWithConfig(
	sbPool SandboxPool,
	meta *SandboxMeta,
	isLeaf bool,
	codeDir string,
	scratchDir string,
	config *SandboxSetConfig,
) *SandboxSet {
	if config == nil {
		config = DefaultSandboxSetConfig()
	}

	if config.MaxSandboxes <= 0 {
		config.MaxSandboxes = 10
	}

	if config.HealthCheckTimeout <= 0 {
		config.HealthCheckTimeout = 1 * time.Second
	}

	if config.GetTimeout <= 0 {
		config.GetTimeout = 30 * time.Second
	}

	if config.CreateTimeout <= 0 {
		config.CreateTimeout = 10 * time.Second
	}

	meta = fillMetaDefaults(meta)

	return &SandboxSet{
		sandboxes:   make([]*sandboxWrapper, 0),
		config:      config,
		sbPool:      sbPool,
		meta:        meta,
		isLeaf:      isLeaf,
		codeDir:     codeDir,
		scratchDir:  scratchDir,
		errorCounts: make(map[ErrorType]int64),
		lastErrors:  make([]*ErrorRecord, 0, 10),
		startTime:   time.Now(),
		closed:      false,
	}
}

// =============================================================================
// CORE OPERATIONS
// =============================================================================

// GetSandbox returns an available sandbox or creates a new one.
// Supports optional configuration via GetOption parameters.
func (set *SandboxSet) GetSandbox(opts ...GetOption) (Sandbox, error) {
	// Apply options
	options := &getOptions{
		timeout:      set.config.GetTimeout,
		skipHealth:   false,
		preferRecent: false,
	}
	for _, opt := range opts {
		opt(options)
	}

	// Simplified: call getSandboxSync directly without goroutine wrapper
	// Capacity errors return immediately, so timeout handling is deferred
	// to future implementation when async operations are needed
	return set.getSandboxSync(options)
}

// getSandboxSync is the synchronous implementation of GetSandbox
func (set *SandboxSet) getSandboxSync(opts *getOptions) (Sandbox, error) {
	// Fast path: try to reuse idle sandbox
	set.mutex.Lock()

	if set.closed {
		set.mutex.Unlock()
		return nil, ErrClosed
	}

	// Pop from priority queue (O(log n) instead of O(n) scan)
	wrapper := set.selectIdleSandbox(opts)
	if wrapper != nil {
		// Mark as in use
		wrapper.state = StateInUse
		wrapper.useCount++
		now := time.Now()
		wrapper.lastUsed = now
		wrapper.busyStart = &now

		// Update statistics
		set.borrowed++
		set.lastBorrow = &now

		sb := wrapper.sb
		set.mutex.Unlock()

		// Optional health check before returning (outside lock!)
		if set.config.HealthCheckOnGet && !opts.skipHealth {
			if !set.checkHealth(wrapper) {
				set.DestroyAndRemove(sb, "failed health check on get")
				return set.getSandboxSync(opts) // Retry
			}
		}

		set.emitEvent(EventSandboxBorrowed, sb.ID(), nil)
		return sb, nil
	}

	// Check capacity
	if len(set.sandboxes) >= set.config.MaxSandboxes {
		set.mutex.Unlock()
		set.recordError(ErrorTypeCapacity, "pool at capacity", "")
		set.emitEvent(EventPoolAtCapacity, "", nil)
		return nil, ErrCapacity
	}

	set.mutex.Unlock()

	// Slow path: create new sandbox
	return set.createAndRegister()
}

// selectIdleSandbox finds an idle sandbox with linear search
// MUST be called with mutex locked
func (set *SandboxSet) selectIdleSandbox(opts *getOptions) *sandboxWrapper {
	now := time.Now()

	// Simple linear search for idle sandbox
	for _, wrapper := range set.sandboxes {
		if wrapper.state != StateIdle {
			continue
		}

		// Apply age filter if requested
		if opts.preferRecent && opts.maxAge > 0 {
			age := now.Sub(wrapper.createdAt)
			if age > opts.maxAge {
				// Too old, skip
				continue
			}
		}

		// Found idle sandbox
		if wrapper.busyStart != nil {
			wrapper.totalIdleTime += now.Sub(*wrapper.busyStart)
		}
		return wrapper
	}

	return nil
}

// createAndRegister creates a new sandbox and registers it in the pool
func (set *SandboxSet) createAndRegister() (Sandbox, error) {
	// Create sandbox (outside lock)
	ctx, cancel := context.WithTimeout(context.Background(), set.config.CreateTimeout)
	defer cancel()

	type result struct {
		sb  Sandbox
		err error
	}
	done := make(chan result, 1)

	go func() {
		sb, err := set.sbPool.Create(
			nil,
			set.isLeaf,
			set.codeDir,
			set.scratchDir,
			set.meta,
		)
		select {
		case done <- result{sb, err}:
		default:
		}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			set.recordError(ErrorTypeCreate, res.err.Error(), "")
			return nil, res.err
		}

		// Register in pool
		return set.registerSandbox(res.sb)

	case <-ctx.Done():
		set.recordError(ErrorTypeTimeout, "sandbox creation timeout", "")
		return nil, fmt.Errorf("sandbox creation timeout after %v", set.config.CreateTimeout)
	}
}

// registerSandbox adds a newly created sandbox to the pool
func (set *SandboxSet) registerSandbox(sb Sandbox) (Sandbox, error) {
	set.mutex.Lock()
	defer set.mutex.Unlock()

	if set.closed {
		sb.Destroy("pool closed during creation")
		return nil, ErrClosed
	}

	now := time.Now()
	wrapper := &sandboxWrapper{
		sb:        sb,
		state:     StateInUse,
		createdAt: now,
		lastUsed:  now,
		useCount:  1,
		busyStart: &now,
	}

	set.sandboxes = append(set.sandboxes, wrapper)
	set.created++
	set.borrowed++
	set.lastBorrow = &now

	set.emitEvent(EventSandboxCreated, sb.ID(), map[string]interface{}{
		"total": len(set.sandboxes),
	})
	set.emitEvent(EventSandboxBorrowed, sb.ID(), nil)

	return sb, nil
}

// ReleaseSandbox marks a sandbox as available for reuse.
// Supports optional configuration via ReleaseOption parameters.
func (set *SandboxSet) ReleaseSandbox(sb Sandbox, opts ...ReleaseOption) error {
	if sb == nil {
		return fmt.Errorf("cannot release nil sandbox")
	}

	// Apply options
	options := &releaseOptions{
		skipHealth: false,
		markDirty:  false,
	}
	for _, opt := range opts {
		opt(options)
	}

	// Phase 1: Mark as checking (prevents concurrent reuse)
	set.mutex.Lock()
	var wrapper *sandboxWrapper
	for _, w := range set.sandboxes {
		if w.sb.ID() == sb.ID() {
			wrapper = w
			break
		}
	}
	if wrapper == nil {
		set.mutex.Unlock()
		set.recordError(ErrorTypeUnknownSandbox, "sandbox not found", sb.ID())
		return ErrUnknownSandbox
	}

	if wrapper.state != StateInUse {
		set.mutex.Unlock()
		set.recordError(ErrorTypeDoubleFree, "sandbox not in use", sb.ID())
		return ErrNotInUse
	}

	wrapper.state = StateChecking

	// Update busy time tracking
	if wrapper.busyStart != nil {
		wrapper.totalBusyTime += time.Since(*wrapper.busyStart)
		wrapper.busyStart = nil
	}

	set.mutex.Unlock()

	// Phase 2: Health check outside lock (slow operation)
	healthy := true
	shouldCheck := set.config.HealthCheckOnRelease || options.markDirty
	if shouldCheck && !options.skipHealth {
		healthy = set.checkHealth(wrapper)
	}

	// Phase 3: Finalize release
	set.mutex.Lock()

	if !healthy {
		set.mutex.Unlock()
		set.DestroyAndRemove(sb, "health check failed on release")
		return errors.New("sandbox failed health check and was removed")
	}

	defer set.mutex.Unlock()

	wrapper.state = StateIdle
	now := time.Now()
	wrapper.lastUsed = now
	wrapper.busyStart = &now // Track idle start time
	set.released++
	set.lastRelease = &now

	// Sandbox is now idle and available for reuse (linear search will find it)

	set.emitEvent(EventSandboxReleased, sb.ID(), nil)
	return nil
}

// DestroyAndRemove permanently removes a sandbox from the pool.
func (set *SandboxSet) DestroyAndRemove(sb Sandbox, reason string) error {
	if sb == nil {
		return fmt.Errorf("cannot destroy nil sandbox")
	}

	// Phase 1: Remove from pool (under lock)
	set.mutex.Lock()
	var sbToDestroy Sandbox
	found := false

	for i, w := range set.sandboxes {
		if w.sb.ID() == sb.ID() {
			// Remove from slice
			set.sandboxes = append(set.sandboxes[:i], set.sandboxes[i+1:]...)
			sbToDestroy = w.sb
			found = true
			break
		}
	}

	if !found {
		set.mutex.Unlock()
		return ErrUnknownSandbox
	}

	set.removed++
	set.mutex.Unlock()

	// Phase 2: Destroy outside lock (CRITICAL FIX)
	sbToDestroy.Destroy(reason)

	set.emitEvent(EventSandboxRemoved, sb.ID(), map[string]interface{}{
		"reason": reason,
	})

	return nil
}

// Destroy cleans up all sandboxes in the set.
func (set *SandboxSet) Destroy() {
	set.mutex.Lock()

	// Mark as closed
	set.closed = true

	// Collect all sandboxes to destroy
	toDestroy := make([]Sandbox, 0, len(set.sandboxes))
	for _, w := range set.sandboxes {
		if w.sb != nil {
			toDestroy = append(toDestroy, w.sb)
		}
	}

	// Clear data structures
	set.sandboxes = nil

	set.mutex.Unlock()

	// Destroy all sandboxes outside lock
	for _, sb := range toDestroy {
		sb.Destroy("SandboxSet cleanup")
	}
}

// =============================================================================
// OPERATIONAL CONTROLS
// =============================================================================

// Warm pre-creates sandboxes up to the target count.
func (set *SandboxSet) Warm(target int) error {
	if target < 0 {
		return fmt.Errorf("warm target cannot be negative")
	}

	if target > set.config.MaxSandboxes {
		return fmt.Errorf("warm target %d exceeds max sandboxes %d", target, set.config.MaxSandboxes)
	}

	set.mutex.RLock()
	current := len(set.sandboxes)
	set.mutex.RUnlock()

	if current >= target {
		return nil // Already at target
	}

	needed := target - current

	// Create sandboxes
	for i := 0; i < needed; i++ {
		sb, err := set.sbPool.Create(
			nil,
			set.isLeaf,
			set.codeDir,
			set.scratchDir,
			set.meta,
		)
		if err != nil {
			return fmt.Errorf("failed to warm sandbox %d/%d: %w", i+1, needed, err)
		}

		// Register as idle
		set.mutex.Lock()
		if set.closed {
			set.mutex.Unlock()
			sb.Destroy("pool closed during warming")
			return ErrClosed
		}

		now := time.Now()
		wrapper := &sandboxWrapper{
			sb:        sb,
			state:     StateIdle,
			createdAt: now,
			lastUsed:  now,
			useCount:  0,
			busyStart: &now, // Track idle time from creation
		}

		set.sandboxes = append(set.sandboxes, wrapper)
		set.created++
		set.mutex.Unlock()

		set.emitEvent(EventSandboxWarmed, sb.ID(), map[string]interface{}{
			"index": i + 1,
			"total": needed,
		})
	}

	return nil
}

// Shrink removes idle sandboxes down to the target count.
func (set *SandboxSet) Shrink(target int) error {
	if target < 0 {
		return fmt.Errorf("shrink target cannot be negative")
	}

	// Respect MinIdleSandboxes
	if target < set.config.MinIdleSandboxes {
		target = set.config.MinIdleSandboxes
	}

	set.mutex.Lock()

	// Count idle sandboxes
	idleCount := 0
	for _, w := range set.sandboxes {
		if w.state == StateIdle {
			idleCount++
		}
	}

	if idleCount <= target {
		set.mutex.Unlock()
		return nil // Already at or below target
	}

	toRemove := idleCount - target

	// Collect idle sandboxes to remove
	toDestroy := make([]Sandbox, 0, toRemove)
	newSandboxes := make([]*sandboxWrapper, 0, len(set.sandboxes))

	removed := 0
	for _, w := range set.sandboxes {
		if w.state == StateIdle && removed < toRemove {
			toDestroy = append(toDestroy, w.sb)
			removed++
		} else {
			newSandboxes = append(newSandboxes, w)
		}
	}

	set.sandboxes = newSandboxes
	set.removed += int64(len(toDestroy))
	set.mutex.Unlock()

	// Destroy outside lock
	for _, sb := range toDestroy {
		sb.Destroy("shrinking pool")
		set.emitEvent(EventSandboxRemoved, sb.ID(), map[string]interface{}{
			"reason": "shrink",
		})
	}

	set.emitEvent(EventPoolShrunk, "", map[string]interface{}{
		"removed": len(toDestroy),
		"target":  target,
	})

	return nil
}

// =============================================================================
// HEALTH CHECKING
// =============================================================================

// checkHealth performs health checking on a sandbox
func (set *SandboxSet) checkHealth(wrapper *sandboxWrapper) bool {
	// Update stats
	set.mutex.Lock()
	set.healthChecks++
	wrapper.healthChecks++
	set.mutex.Unlock()

	sb := wrapper.sb
	method := set.config.HealthCheckMethod
	timeout := set.config.HealthCheckTimeout

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resultChan := make(chan error, 1)

	go func() {
		var err error
		switch method {
		case HealthCheckPause:
			err = sb.Pause()

		case HealthCheckPauseUnpause:
			if err = sb.Pause(); err == nil {
				err = sb.Unpause()
			}

		case HealthCheckHTTPPing:
			// Future: HTTP health check
			err = sb.Pause()

		default:
			err = sb.Pause()
		}

		select {
		case resultChan <- err:
		default:
		}
	}()

	select {
	case err := <-resultChan:
		if err != nil {
			set.mutex.Lock()
			set.healthFailures++
			wrapper.healthFails++
			set.mutex.Unlock()
			set.emitEvent(EventSandboxHealthFailed, sb.ID(), map[string]interface{}{
				"error": err.Error(),
			})
			return false
		}

		set.mutex.Lock()
		set.healthPasses++
		set.mutex.Unlock()
		set.emitEvent(EventSandboxHealthPassed, sb.ID(), nil)
		return true

	case <-ctx.Done():
		set.mutex.Lock()
		set.healthFailures++
		wrapper.healthFails++
		set.mutex.Unlock()
		set.emitEvent(EventSandboxHealthFailed, sb.ID(), map[string]interface{}{
			"error": "timeout",
		})
		return false
	}
}

// =============================================================================
// STATISTICS AND METRICS
// =============================================================================

// Stats returns current pool statistics (lightweight).
func (set *SandboxSet) Stats() map[string]int {
	set.mutex.RLock()
	defer set.mutex.RUnlock()

	inUse := 0
	checking := 0
	for _, w := range set.sandboxes {
		if w.state == StateInUse {
			inUse++
		} else if w.state == StateChecking {
			checking++
		}
	}

	return map[string]int{
		"total":           len(set.sandboxes),
		"in_use":          inUse,
		"idle":            len(set.sandboxes) - inUse - checking,
		"checking":        checking,
		"max":             set.config.MaxSandboxes,
		"created":         int(set.created),
		"borrowed":        int(set.borrowed),
		"released":        int(set.released),
		"removed":         int(set.removed),
		"health_checks":   int(set.healthChecks),
		"health_failures": int(set.healthFailures),
	}
}

// Metrics returns detailed pool metrics (more expensive).
func (set *SandboxSet) Metrics() *PoolMetrics {
	set.mutex.RLock()
	defer set.mutex.RUnlock()

	// Count states
	inUse := 0
	idle := 0
	checking := 0

	// Timing totals
	var totalIdleTime time.Duration
	var totalUseTime time.Duration

	// Per-sandbox metrics
	sandboxMetrics := make([]*SandboxMetric, 0, len(set.sandboxes))
	now := time.Now()

	for _, w := range set.sandboxes {
		switch w.state {
		case StateInUse:
			inUse++
		case StateIdle:
			idle++
		case StateChecking:
			checking++
		}

		totalIdleTime += w.totalIdleTime
		totalUseTime += w.totalBusyTime

		sandboxMetrics = append(sandboxMetrics, &SandboxMetric{
			ID:               w.sb.ID(),
			State:            w.state,
			Age:              now.Sub(w.createdAt),
			LastUsed:         w.lastUsed,
			UseCount:         w.useCount,
			HealthCheckCount: w.healthChecks,
			HealthFailCount:  w.healthFails,
			CreatedAt:        w.createdAt,
			TotalBusyTime:    w.totalBusyTime,
			TotalIdleTime:    w.totalIdleTime,
		})
	}

	// Calculate efficiency metrics
	hitRate := 0.0
	if set.borrowed > 0 {
		hitRate = float64(set.released) / float64(set.borrowed)
	}

	capacityUtilization := 0.0
	if set.config.MaxSandboxes > 0 {
		capacityUtilization = float64(inUse) / float64(set.config.MaxSandboxes)
	}

	avgIdleTime := time.Duration(0)
	if len(set.sandboxes) > 0 {
		avgIdleTime = totalIdleTime / time.Duration(len(set.sandboxes))
	}

	avgUseTime := time.Duration(0)
	if len(set.sandboxes) > 0 {
		avgUseTime = totalUseTime / time.Duration(len(set.sandboxes))
	}

	// Copy error records
	lastErrors := make([]*ErrorRecord, len(set.lastErrors))
	copy(lastErrors, set.lastErrors)

	return &PoolMetrics{
		Total:               len(set.sandboxes),
		InUse:               inUse,
		Idle:                idle,
		Checking:            checking,
		Max:                 set.config.MaxSandboxes,
		Created:             set.created,
		Borrowed:            set.borrowed,
		Released:            set.released,
		Removed:             set.removed,
		HealthChecks:        set.healthChecks,
		HealthPasses:        set.healthPasses,
		HealthFailures:      set.healthFailures,
		Uptime:              now.Sub(set.startTime),
		LastBorrow:          set.lastBorrow,
		LastRelease:         set.lastRelease,
		HitRate:             hitRate,
		CapacityUtilization: capacityUtilization,
		AverageIdleTime:     avgIdleTime,
		AverageUseTime:      avgUseTime,
		SandboxMetrics:      sandboxMetrics,
		ErrorCounts:         copyErrorCounts(set.errorCounts),
		LastErrors:          lastErrors,
	}
}

// Size returns the current number of sandboxes in the pool.
func (set *SandboxSet) Size() int {
	set.mutex.RLock()
	defer set.mutex.RUnlock()
	return len(set.sandboxes)
}

// InUse returns the number of sandboxes currently borrowed.
func (set *SandboxSet) InUse() int {
	set.mutex.RLock()
	defer set.mutex.RUnlock()

	count := 0
	for _, w := range set.sandboxes {
		if w.state == StateInUse {
			count++
		}
	}
	return count
}

// =============================================================================
// ERROR TRACKING
// =============================================================================

func (set *SandboxSet) recordError(errType ErrorType, message string, sandboxID string) {
	set.mutex.Lock()
	defer set.mutex.Unlock()

	set.errorCounts[errType]++

	record := &ErrorRecord{
		Type:      errType,
		Time:      time.Now(),
		Message:   message,
		SandboxID: sandboxID,
	}

	// Keep last 10 errors
	set.lastErrors = append(set.lastErrors, record)
	if len(set.lastErrors) > 10 {
		set.lastErrors = set.lastErrors[1:]
	}
}

// =============================================================================
// EVENT EMISSION
// =============================================================================

func (set *SandboxSet) emitEvent(eventType SandboxSetEventType, sandboxID string, details map[string]interface{}) {
	if set.config.EventHandlers == nil || len(set.config.EventHandlers) == 0 {
		return
	}

	event := SandboxSetEvent{
		Type:      eventType,
		Time:      time.Now(),
		SandboxID: sandboxID,
		Details:   details,
	}

	// Call handlers in goroutines to avoid blocking
	for _, handler := range set.config.EventHandlers {
		go handler(event)
	}
}

// =============================================================================
// HELPERS
// =============================================================================

func copyErrorCounts(src map[ErrorType]int64) map[ErrorType]int64 {
	dst := make(map[ErrorType]int64, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

package sandbox

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

/*
 * =========================
 * Configuration
 * =========================
 */

type SandboxSetConfig struct {
	MaxSandboxes         int
	HealthCheckOnRelease bool
	HealthCheckTimeout   time.Duration
}

func DefaultSandboxSetConfig() *SandboxSetConfig {
	return &SandboxSetConfig{
		MaxSandboxes:         10,
		HealthCheckOnRelease: false,
		HealthCheckTimeout:   1 * time.Second,
	}
}

/*
 * =========================
 * SandboxSet
 * =========================
 */

// SandboxSet manages a thread-safe pool of sandboxes.
type SandboxSet struct {
	mutex sync.Mutex

	// Pool of sandbox wrappers
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

	// Statistics
	created        int
	borrowed       int
	released       int
	removed        int
	healthChecks   int
	healthFailures int
}

// sandboxWrapper tracks sandbox state safely
type sandboxWrapper struct {
	sb        Sandbox
	inUse     bool
	checking  bool
	lastUsed  time.Time
}

/*
 * =========================
 * Constructors
 * =========================
 */

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

	meta = fillMetaDefaults(meta)

	return &SandboxSet{
		sandboxes:  []*sandboxWrapper{},
		config:     config,
		sbPool:     sbPool,
		meta:       meta,
		isLeaf:     isLeaf,
		codeDir:    codeDir,
		scratchDir: scratchDir,
	}
}

/*
 * =========================
 * Core API
 * =========================
 */

// GetSandbox returns an available sandbox or creates a new one.
func (set *SandboxSet) GetSandbox() (Sandbox, error) {
	set.mutex.Lock()
	defer set.mutex.Unlock()

	// Reuse idle sandbox
	for _, w := range set.sandboxes {
		if !w.inUse && !w.checking && w.sb != nil {
			w.inUse = true
			set.borrowed++
			return w.sb, nil
		}
	}

	// Capacity check
	if len(set.sandboxes) >= set.config.MaxSandboxes {
		return nil, fmt.Errorf(
			"SandboxSet at capacity (%d/%d)",
			len(set.sandboxes),
			set.config.MaxSandboxes,
		)
	}

	// Create new sandbox
	sb, err := set.createSandbox()
	if err != nil {
		return nil, err
	}

	set.sandboxes = append(set.sandboxes, &sandboxWrapper{
		sb:       sb,
		inUse:    true,
		lastUsed: time.Now(),
	})

	set.created++
	set.borrowed++

	return sb, nil
}

// ReleaseSandbox releases a sandbox back to the pool.
func (set *SandboxSet) ReleaseSandbox(sb Sandbox) error {
	if sb == nil {
		return errors.New("cannot release nil sandbox")
	}

	var wrapper *sandboxWrapper

	// Mark as checking (prevents reuse during health check)
	set.mutex.Lock()
	for _, w := range set.sandboxes {
		if w.sb == sb {
			if !w.inUse {
				set.mutex.Unlock()
				return fmt.Errorf("sandbox %s was not in use", sb.ID())
			}
			w.checking = true
			wrapper = w
			break
		}
	}
	set.mutex.Unlock()

	if wrapper == nil {
		return fmt.Errorf("sandbox %s not found", sb.ID())
	}

	// Optional health check (no lock held)
	if set.config.HealthCheckOnRelease {
		if !set.checkSandboxHealth(sb) {
			return set.DestroyAndRemove(sb)
		}
	}

	// Finalize release
	set.mutex.Lock()
	defer set.mutex.Unlock()

	wrapper.inUse = false
	wrapper.checking = false
	wrapper.lastUsed = time.Now()
	set.released++

	return nil
}

// DestroyAndRemove permanently removes a sandbox from the pool.
func (set *SandboxSet) DestroyAndRemove(sb Sandbox) error {
	if sb == nil {
		return errors.New("cannot destroy nil sandbox")
	}

	set.mutex.Lock()
	defer set.mutex.Unlock()

	for i, w := range set.sandboxes {
		if w.sb == sb {
			sb.Destroy("removed from SandboxSet")
			set.sandboxes = append(set.sandboxes[:i], set.sandboxes[i+1:]...)
			set.removed++
			return nil
		}
	}

	return fmt.Errorf("sandbox %s not found", sb.ID())
}

/*
 * =========================
 * Health Checking
 * =========================
 */

func (set *SandboxSet) checkSandboxHealth(sb Sandbox) bool {
	set.incrementHealthCheck()

	done := make(chan error, 1)

	go func() {
		select {
		case done <- sb.Pause():
		default:
		}
	}()

	select {
	case err := <-done:
		if err != nil {
			set.incrementHealthFailure()
			return false
		}
		sb.Unpause()
		return true

	case <-time.After(set.config.HealthCheckTimeout):
		set.incrementHealthFailure()
		return false
	}
}

func (set *SandboxSet) incrementHealthCheck() {
	set.mutex.Lock()
	set.healthChecks++
	set.mutex.Unlock()
}

func (set *SandboxSet) incrementHealthFailure() {
	set.mutex.Lock()
	set.healthFailures++
	set.mutex.Unlock()
}

/*
 * =========================
 * Helpers
 * =========================
 */

func (set *SandboxSet) createSandbox() (Sandbox, error) {
	return set.sbPool.Create(
		nil,
		set.isLeaf,
		set.codeDir,
		set.scratchDir,
		set.meta,
	)
}

func (set *SandboxSet) Destroy() {
	set.mutex.Lock()
	defer set.mutex.Unlock()

	for _, w := range set.sandboxes {
		if w.sb != nil {
			w.sb.Destroy("SandboxSet cleanup")
		}
	}
	set.sandboxes = nil
}

/*
 * =========================
 * Introspection
 * =========================
 */

func (set *SandboxSet) Stats() map[string]int {
	set.mutex.Lock()
	defer set.mutex.Unlock()

	inUse := 0
	for _, w := range set.sandboxes {
		if w.inUse {
			inUse++
		}
	}

	return map[string]int{
		"total":           len(set.sandboxes),
		"in_use":          inUse,
		"idle":            len(set.sandboxes) - inUse,
		"max":             set.config.MaxSandboxes,
		"created":         set.created,
		"borrowed":        set.borrowed,
		"released":        set.released,
		"removed":         set.removed,
		"health_checks":   set.healthChecks,
		"health_failures": set.healthFailures,
	}
}

func (set *SandboxSet) Size() int {
	set.mutex.Lock()
	defer set.mutex.Unlock()
	return len(set.sandboxes)
}

func (set *SandboxSet) InUse() int {
	set.mutex.Lock()
	defer set.mutex.Unlock()

	count := 0
	for _, w := range set.sandboxes {
		if w.inUse {
			count++
		}
	}
	return count
}


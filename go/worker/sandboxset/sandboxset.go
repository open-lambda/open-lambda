package sandboxset

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// SandboxRef is a handle returned by GetOrCreateUnpaused.
// It wraps a sandbox with a back-pointer to its parent set.
// Set Broken = true before calling Put if the sandbox should not be recycled.
type SandboxRef struct {
	sb        sandbox.Sandbox
	set       *sandboxSetImpl
	Broken    bool        // public: caller sets true if request failed; Put will destroy instead of recycle
	inUse     bool        // true when checked out; false when idle in pool
	destroyed atomic.Bool // set atomically after Destroy(); guards against concurrent Close + Put
}

// Sandbox returns the underlying sandbox.
func (r *SandboxRef) Sandbox() sandbox.Sandbox {
	return r.sb
}

// Put returns the sandbox to its parent set, or destroys it if Broken is true.
func (r *SandboxRef) Put() error {
	if r.destroyed.Load() {
		return fmt.Errorf("sandboxset: sandbox %s already destroyed", r.sb.ID())
	}
	if r.Broken {
		return r.set.destroy(r, "state marked broken")
	}
	return r.set.put(r)
}

// Destroy removes the sandbox from its parent set and destroys it.
func (r *SandboxRef) Destroy(reason string) error {
	if r.destroyed.Load() {
		return nil
	}
	return r.set.destroy(r, reason)
}

// sandboxSetImpl is the private concrete type returned by New.
// All mutable state is guarded by mu.
type sandboxSetImpl struct {
	mu     sync.Mutex
	pool   []*SandboxRef
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

// claimIdle returns an idle ref from the pool, marking it inUse.
// Caller must hold s.mu.
func (s *sandboxSetImpl) claimIdle() *SandboxRef {
	for _, ref := range s.pool {
		if !ref.inUse {
			ref.inUse = true
			return ref
		}
	}
	return nil
}

// tryClaimIdle acquires the lock for its full duration, checks closed,
// and returns an idle ref (or nil if none available).
func (s *sandboxSetImpl) tryClaimIdle() (*SandboxRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, fmt.Errorf("sandboxset: closed")
	}
	return s.claimIdle(), nil
}

// GetOrCreateUnpaused implements SandboxSet.
//
// Fast path: claim an idle ref via tryClaimIdle (which holds the lock for
// its full duration), then Unpause outside the lock so the pool is not
// stalled during I/O.
//
// Slow path: no idle ref exists, so a new sandbox is created without
// holding the lock.
func (s *sandboxSetImpl) GetOrCreateUnpaused() (*SandboxRef, error) {
	for {
		claimed, err := s.tryClaimIdle()
		if err != nil {
			return nil, err
		}
		if claimed == nil {
			break // no idle sandbox — fall through to Create
		}

		// Unpause outside the lock.
		if err := claimed.sb.Unpause(); err != nil {
			_ = s.destroy(claimed, fmt.Sprintf("unpause: %v", err))
			continue // try the next idle sandbox
		}
		return claimed, nil
	}

	// Slow path: create a new sandbox without holding the lock.
	var parentSb sandbox.Sandbox
	if s.cfg.Parent != nil {
		parentRef, err := s.cfg.Parent.GetOrCreateUnpaused()
		if err != nil {
			return nil, fmt.Errorf("sandboxset: parent get: %w", err)
		}
		parentSb = parentRef.Sandbox()
		defer parentRef.Put()
	}

	scratchDir := s.cfg.ScratchDirs.Make("sb")
	sb, err := s.cfg.Pool.Create(
		parentSb, s.cfg.IsLeaf,
		s.cfg.CodeDir, scratchDir,
		s.cfg.Meta,
	)
	if err != nil {
		return nil, fmt.Errorf("sandboxset: create: %w", err)
	}

	ref := &SandboxRef{sb: sb, set: s, inUse: true}
	if !s.appendToPool(ref) {
		sb.Destroy("set closed during create")
		return nil, fmt.Errorf("sandboxset: closed")
	}
	return ref, nil
}

// appendToPool appends ref to the pool if the set is not closed.
// Returns false if the set is closed. Caller must not hold s.mu.
func (s *sandboxSetImpl) appendToPool(ref *SandboxRef) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.pool = append(s.pool, ref)
	return true
}

// put pauses the sandbox and returns it to the idle pool.
// Pause is called before acquiring the lock to avoid blocking pool access during I/O.
func (s *sandboxSetImpl) put(ref *SandboxRef) error {
	if err := ref.sb.Pause(); err != nil {
		_ = s.destroy(ref, fmt.Sprintf("pause failed: %v", err))
		return fmt.Errorf("sandboxset: sandbox %s destroyed because Pause failed: %w", ref.sb.ID(), err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("sandboxset: closed (sandbox %s was destroyed by Close)", ref.sb.ID())
	}

	ref.inUse = false
	return nil
}

// tryRemoveFromPool removes ref from the pool using O(1) swap-with-tail.
// Returns true if found. Caller must not hold s.mu.
func (s *sandboxSetImpl) tryRemoveFromPool(ref *SandboxRef) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.pool {
		if r == ref {
			s.pool[i] = s.pool[len(s.pool)-1]
			s.pool = s.pool[:len(s.pool)-1]
			return true
		}
	}
	return false
}

// destroy removes ref from the pool and destroys the underlying sandbox.
// The sandbox is always destroyed even if not found in the pool.
func (s *sandboxSetImpl) destroy(ref *SandboxRef, reason string) error {
	found := s.tryRemoveFromPool(ref)
	ref.sb.Destroy(reason) // I/O outside lock
	ref.destroyed.Store(true)
	if !found {
		return fmt.Errorf("sandboxset: sandbox %s not found in pool (still destroyed)", ref.sb.ID())
	}
	return nil
}

// Close implements SandboxSet.
func (s *sandboxSetImpl) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("sandboxset: already closed")
	}
	s.closed = true

	for _, ref := range s.pool {
		ref.sb.Destroy("sandboxset closed")
		ref.destroyed.Store(true)
	}
	s.pool = nil
	return nil
}

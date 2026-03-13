package sandboxset

import (
	"fmt"
	"sync"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// SandboxState describes the health of a checked-out sandbox.
type SandboxState int

const (
	StateReady  SandboxState = iota // healthy, usable
	StateBroken                     // error occurred, should be destroyed
)

// SandboxRef is a handle returned by GetOrCreateUnpaused.
// It wraps a sandbox together with a back-pointer to its parent set
// and a health state, so the caller can Put or Destroy without
// tracking which set the sandbox came from.
type SandboxRef struct {
	sb    sandbox.Sandbox
	set   *sandboxSetImpl
	State SandboxState
}

// Sandbox returns the underlying sandbox.
func (r *SandboxRef) Sandbox() sandbox.Sandbox {
	return r.sb
}

// Put returns the sandbox to its parent set.
// This is a convenience method equivalent to set.Put(ref.Sandbox()).
func (r *SandboxRef) Put() error {
	return r.set.Put(r.sb)
}

// Destroy removes the sandbox from its parent set and destroys it.
// This is a convenience method equivalent to set.Destroy(ref.Sandbox(), reason).
func (r *SandboxRef) Destroy(reason string) error {
	return r.set.Destroy(r.sb, reason)
}

// sandboxWrapper pairs a sandbox with an in-use flag.
type sandboxWrapper struct {
	sb    sandbox.Sandbox
	inUse bool
}

// sandboxSetImpl is the private concrete type returned by New.
// All mutable state is guarded by mu.
type sandboxSetImpl struct {
	mu     sync.Mutex
	pool   []*sandboxWrapper
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

// makeScratchDir creates a scratch directory for a new sandbox.
// DirMaker.Make panics on failure (e.g., disk full), so we recover
// here and return an error instead of crashing the worker.
func (s *sandboxSetImpl) makeScratchDir() (dir string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	dir = s.cfg.ScratchDirs.Make("sb")
	return dir, nil
}

// GetOrCreateUnpaused implements SandboxSet.
//
// Fast path: an idle sandbox is claimed under a short lock, then Unpause
// runs outside the lock so the pool is not stalled during I/O.
//
// Slow path: no idle sandbox exists, so a new one is created without
// holding the lock.
func (s *sandboxSetImpl) GetOrCreateUnpaused() (*SandboxRef, error) {
	// Loop over idle sandboxes until one unpauses successfully,
	// or the pool has no idle sandboxes left.
	for {
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return nil, fmt.Errorf("sandboxset: closed")
		}

		// Fast path: claim an idle sandbox.
		var claimed sandbox.Sandbox // raw sandbox, wrapped in SandboxRef on return
		for _, w := range s.pool {
			if !w.inUse {
				w.inUse = true
				claimed = w.sb
				break
			}
		}
		s.mu.Unlock()

		if claimed == nil {
			break // no idle sandbox — fall through to Create
		}

		// Unpause outside the lock (split-lock pattern).
		if err := claimed.Unpause(); err != nil {
			_ = s.Destroy(claimed, fmt.Sprintf("unpause: %v", err))
			continue // try the next idle sandbox
		}
		return &SandboxRef{sb: claimed, set: s, State: StateReady}, nil
	}

	// Slow path: create a new sandbox without holding the lock.
	scratchDir, err := s.makeScratchDir()
	if err != nil {
		return nil, fmt.Errorf("sandboxset: scratch dir: %w", err)
	}
	sb, err := s.cfg.Pool.Create(
		s.cfg.Parent, s.cfg.IsLeaf,
		s.cfg.CodeDir, scratchDir,
		s.cfg.Meta,
	)
	if err != nil {
		return nil, fmt.Errorf("sandboxset: create: %w", err)
	}

	s.mu.Lock()
	s.pool = append(s.pool, &sandboxWrapper{sb: sb, inUse: true})
	s.mu.Unlock()

	return &SandboxRef{sb: sb, set: s, State: StateReady}, nil
}

// Put implements SandboxSet.
//
// The sandbox is paused and its wrapper is flipped back to idle.
// If Pause fails, the sandbox is destroyed rather than silently
// recycled — a bad sandbox should never re-enter the pool.
func (s *sandboxSetImpl) Put(sb sandbox.Sandbox) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("sandboxset: closed (sandbox %s was destroyed by Close)", sb.ID())
	}
	s.mu.Unlock()

	if err := sb.Pause(); err != nil {
		_ = s.Destroy(sb, fmt.Sprintf("pause failed: %v", err))
		return fmt.Errorf("sandboxset: sandbox %s destroyed because Pause failed: %w", sb.ID(), err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, w := range s.pool {
		if w.sb.ID() == sb.ID() {
			w.inUse = false
			return nil
		}
	}
	return fmt.Errorf("sandboxset: sandbox %s not found in pool", sb.ID())
}

// Destroy implements SandboxSet.
//
// The wrapper is spliced out of the pool under a short lock using O(1)
// swap-with-tail. Destroy is called outside the lock to keep critical
// sections short. The sandbox is always destroyed even if it was not
// found in the pool.
func (s *sandboxSetImpl) Destroy(sb sandbox.Sandbox, reason string) error {
	s.mu.Lock()
	found := false
	for i, w := range s.pool {
		if w.sb.ID() == sb.ID() {
			s.pool[i] = s.pool[len(s.pool)-1]
			s.pool = s.pool[:len(s.pool)-1]
			found = true
			break
		}
	}
	s.mu.Unlock()

	sb.Destroy(reason)

	if !found {
		return fmt.Errorf("sandboxset: sandbox %s not found in pool (still destroyed)", sb.ID())
	}
	return nil
}

// Close implements SandboxSet.
//
// All sandboxes are snapshot under the lock, then destroyed outside it.
func (s *sandboxSetImpl) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("sandboxset: already closed")
	}
	s.closed = true
	pool := s.pool
	s.pool = nil
	s.mu.Unlock()

	for _, w := range pool {
		w.sb.Destroy("sandboxset closed")
	}
	return nil
}

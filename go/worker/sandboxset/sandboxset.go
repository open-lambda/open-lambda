package sandboxset

import (
	"fmt"
	"sync"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// SandboxRef is a handle returned by GetOrCreateUnpaused.
// Set Broken = true before calling Put if the sandbox should not be recycled.
// sb == nil means the ref has no live sandbox (destroyed or not yet created).

type SandboxRef struct {
	set    *sandboxSetImpl
	Broken bool

	// protected by set.mu
	sb    sandbox.Sandbox
	inUse bool
}

// Sandbox returns the underlying sandbox.
func (r *SandboxRef) Sandbox() sandbox.Sandbox {
	return r.sb
}

// Put returns the sandbox to its parent set, or destroys it if Broken is true.
func (r *SandboxRef) Put() error {
	r.set.mu.Lock()
	already := r.sb == nil
	r.set.mu.Unlock()
	if already {
		return fmt.Errorf("sandboxset: sandbox already destroyed")
	}
	if r.Broken {
		return r.set.destroy(r, "state marked broken")
	}
	return r.set.put(r)
}

// sandboxSetImpl is the private concrete type returned by New.
type sandboxSetImpl struct {
	cfg *Config

	mu     sync.Mutex // protects below fields
	pool   []*SandboxRef
	closed bool
}

func newSandboxSet(cfg *Config) *sandboxSetImpl {
	if cfg == nil {
		panic("sandboxset: Config must not be nil")
	}
	if cfg.Pool == nil {
		panic("sandboxset: Config.Pool must not be nil")
	}
	if cfg.CodeDir == "" {
		panic("sandboxset: Config.CodeDir must not be empty")
	}
	if cfg.ScratchDirs == nil {
		panic("sandboxset: Config.ScratchDirs must not be nil")
	}
	return &sandboxSetImpl{cfg: cfg}
}

// claimIdle acquires the lock and returns an inUse ref.
// It prefers a ref with an existing sandbox, falls back to an empty ref,
// and appends a new ref to the pool if none are idle.
// Always returns a non-nil ref or an error.
func (s *sandboxSetImpl) claimIdle() (*SandboxRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, fmt.Errorf("sandboxset: closed")
	}
	// Path 1: prefer a ref with an existing sandbox.
	for _, ref := range s.pool {
		if !ref.inUse && ref.sb != nil {
			ref.inUse = true
			return ref, nil
		}
	}
	// Path 2: fall back to a ref without a sandbox.
	for _, ref := range s.pool {
		if !ref.inUse {
			ref.inUse = true
			return ref, nil
		}
	}
	// Path 3: no idle ref — create and append a new one.
	ref := &SandboxRef{set: s, inUse: true}
	s.pool = append(s.pool, ref)
	return ref, nil
}

// createSandbox creates a new underlying sandbox, handling parent forking if configured.
// Must be called without holding s.mu.
func (s *sandboxSetImpl) createSandbox() (sandbox.Sandbox, error) {
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
		return nil, err
	}
	return sb, nil
}

// GetOrCreateUnpaused implements SandboxSet.
// Step 1: claim a ref from the pool (requires locking).
// Step 2: ensure the ref has a healthy, unpaused sandbox (no locking).
func (s *sandboxSetImpl) GetOrCreateUnpaused() (*SandboxRef, error) {
	// Step 1: get a SandboxRef (with or without a Sandbox).
	ref, err := s.claimIdle()
	if err != nil {
		return nil, err
	}

	// Step 2: make sure the ref has a healthy, unpaused Sandbox.
	if ref.sb != nil {
		if err := ref.sb.Unpause(); err != nil {
			_ = s.destroy(ref, fmt.Sprintf("unpause: %v", err))
			return nil, fmt.Errorf("sandboxset: unpause: %w", err)
		}
		return ref, nil
	}

	newSb, err := s.createSandbox()
	if err != nil {
		s.mu.Lock()
		ref.inUse = false
		s.mu.Unlock()
		return nil, fmt.Errorf("sandboxset: create: %w", err)
	}

	ref.sb = newSb
	return ref, nil
}

// put pauses the sandbox and returns the ref to the idle pool.
// Pause is called before re-acquiring the lock to avoid blocking pool access during I/O.
func (s *sandboxSetImpl) put(ref *SandboxRef) error {
	s.mu.Lock()
	sb := ref.sb
	s.mu.Unlock()

	if sb == nil {
		return fmt.Errorf("sandboxset: sandbox already destroyed")
	}
	if err := sb.Pause(); err != nil {
		_ = s.destroy(ref, fmt.Sprintf("pause failed: %v", err))
		return fmt.Errorf("sandboxset: sandbox destroyed because Pause failed: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("sandboxset: closed (sandbox destroyed by Close)")
	}

	ref.inUse = false
	return nil
}

// destroy nils out ref.sb under the lock and destroys the underlying sandbox outside it.
// The ref remains in the pool as an idle nil ref, ready to receive a new sandbox.
func (s *sandboxSetImpl) destroy(ref *SandboxRef, reason string) error {
	s.mu.Lock()
	sb := ref.sb
	ref.sb = nil
	ref.inUse = false
	s.mu.Unlock()

	if sb == nil {
		return fmt.Errorf("sandboxset: sandbox already destroyed")
	}
	sb.Destroy(reason) // I/O outside lock
	return nil
}

// Close implements SandboxSet.
func (s *sandboxSetImpl) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return fmt.Errorf("sandboxset: already closed")
	}
	s.closed = true

	var toDestroy []sandbox.Sandbox
	for _, ref := range s.pool {
		if ref.sb != nil {
			toDestroy = append(toDestroy, ref.sb)
			ref.sb = nil
		}
	}
	s.pool = nil
	s.mu.Unlock()

	for _, sb := range toDestroy {
		sb.Destroy("sandboxset closed")
	}
	return nil
}

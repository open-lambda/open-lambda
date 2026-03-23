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

// Destroy removes the sandbox from its parent set and destroys it.
func (r *SandboxRef) Destroy(reason string) error {
	r.set.mu.Lock()
	already := r.sb == nil
	r.set.mu.Unlock()
	if already {
		return nil
	}
	return r.set.destroy(r, reason)
}

// sandboxSetImpl is the private concrete type returned by New.
type sandboxSetImpl struct {
	cfg *Config

	mu     sync.Mutex // protects below fields
	pool   []*SandboxRef
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
// Prefers refs with an existing sandbox (avoids a Create), falls back to nil refs.
// Caller must hold s.mu.
func (s *sandboxSetImpl) claimIdle() *SandboxRef {
	var nilRef *SandboxRef
	for _, ref := range s.pool {
		if ref.inUse {
			continue
		}
		if ref.sb != nil {
			ref.inUse = true
			return ref
		}
		if nilRef == nil {
			nilRef = ref
		}
	}
	if nilRef != nil {
		nilRef.inUse = true
	}
	return nilRef
}

// tryClaimIdle acquires the lock, checks closed, and returns an idle ref (or nil if none).
func (s *sandboxSetImpl) tryClaimIdle() (*SandboxRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, fmt.Errorf("sandboxset: closed")
	}
	return s.claimIdle(), nil
}

// appendNilRef adds a new nil-sb ref to the pool (inUse=true) and returns it.
// Returns an error if the set is closed.
func (s *sandboxSetImpl) appendNilRef() (*SandboxRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, fmt.Errorf("sandboxset: closed")
	}
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
// Fast path: claim an idle ref with an existing sandbox via tryClaimIdle, then Unpause outside the lock.
// Slow path (nil ref): the claimed ref has no sandbox (either destroyed or newly added). A new sandbox is created outside the lock and assigned to the ref.
func (s *sandboxSetImpl) GetOrCreateUnpaused() (*SandboxRef, error) {
	for {
		ref, err := s.tryClaimIdle()
		if err != nil {
			return nil, err
		}

		if ref == nil {
			// No idle ref — add a new nil ref to the pool.
			ref, err = s.appendNilRef()
			if err != nil {
				return nil, err
			}
		}

		s.mu.Lock()
		sb := ref.sb
		s.mu.Unlock()

		if sb != nil {
			// Path 1: existing paused sandbox — just unpause.
			if err := sb.Unpause(); err != nil {
				_ = s.destroy(ref, fmt.Sprintf("unpause: %v", err))
				continue
			}
			return ref, nil
		}

		// Path 2/3: nil ref — create a new sandbox for it.
		newSb, err := s.createSandbox()
		if err != nil {
			s.mu.Lock()
			ref.inUse = false // release back as idle nil ref
			s.mu.Unlock()
			return nil, fmt.Errorf("sandboxset: create: %w", err)
		}

		s.mu.Lock()
		ref.sb = newSb
		s.mu.Unlock()
		return ref, nil
	}
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

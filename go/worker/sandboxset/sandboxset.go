package sandboxset

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// ErrClosed is returned by operations on a closed SandboxSet. Match with errors.Is.
var ErrClosed = errors.New("sandboxset: closed")

// SandboxRef is a handle returned by GetOrCreateUnpaused. While inUse is true
// the holder owns sb; otherwise sb and inUse are protected by set.mu.
// Callers signal a dead sandbox by calling MarkDead before Put. The set never
// destroys sandboxes — lifecycle is owned upstream.
// A ref must not be shared across goroutines; one goroutine holds it at a time.
// Guards that lock s.mu to panic on !inUse use the lock only for the inUse
// check, not to protect sb — sb access is governed by the single-owner rule.
type SandboxRef struct {
	set   *sandboxSetImpl
	sb    sandbox.Sandbox
	inUse bool
}

// Sandbox returns the underlying sandbox. No inUse guard: hot path; misuse is caught by Put/MarkDead guards instead.
func (r *SandboxRef) Sandbox() sandbox.Sandbox { return r.sb }

func (r *SandboxRef) MarkDead() {
	r.set.mu.Lock()
	defer r.set.mu.Unlock()
	if !r.inUse {
		panic(fmt.Sprintf("sandboxset: MarkDead on ref %p not currently held (inUse=%v)", r, r.inUse))
	}
	r.sb = nil
}

func (r *SandboxRef) Put() {
	if r.sb == nil {
		r.set.releaseSlot(r)
	} else {
		r.set.put(r)
	}
}

type sandboxSetImpl struct {
	cfg *Config

	mu     sync.Mutex
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

func (s *sandboxSetImpl) claimIdle() (*SandboxRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, fmt.Errorf("claimIdle: %w", ErrClosed)
	}

	var empty *SandboxRef
	for _, ref := range s.pool {
		if ref.inUse {
			continue
		}
		if ref.sb != nil {
			ref.inUse = true
			return ref, nil
		}
		if empty == nil {
			empty = ref
		}
	}

	if empty != nil {
		empty.inUse = true
		return empty, nil
	}

	ref := &SandboxRef{set: s, inUse: true}
	s.pool = append(s.pool, ref)
	return ref, nil
}

// createSandbox must be called without holding s.mu.
func (s *sandboxSetImpl) createSandbox() (sandbox.Sandbox, error) {
	var parentSb sandbox.Sandbox
	if s.cfg.Parent != nil {
		parentRef, err := s.cfg.Parent.GetOrCreateUnpaused()
		if err != nil {
			return nil, err
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
		return nil, fmt.Errorf("sandboxset: pool create: %w", err)
	}
	return sb, nil
}

func (s *sandboxSetImpl) GetOrCreateUnpaused() (*SandboxRef, error) {
	ref, err := s.claimIdle()
	if err != nil {
		return nil, err
	}

	if ref.sb != nil {
		if err := ref.sb.Unpause(); err != nil {
			slog.Warn("sandboxset: unpause failed, discarding sandbox", "err", err)
			ref.sb = nil
		}
	}

	if ref.sb == nil {
		newSb, err := s.createSandbox()
		if err != nil {
			s.releaseSlot(ref)
			return nil, err
		}
		ref.sb = newSb
	}

	return ref, nil
}

// put relies on Sandbox.Pause being no-op-safe after external death (see sandbox/api.go).
func (s *sandboxSetImpl) put(ref *SandboxRef) {
	s.mu.Lock()
	if !ref.inUse {
		s.mu.Unlock()
		panic(fmt.Sprintf("sandboxset: put on ref %p not currently held (inUse=%v)", ref, ref.inUse))
	}
	closed := s.closed
	if closed {
		s.releaseSlotLocked(ref)
	}
	s.mu.Unlock()
	if closed {
		return
	}

	if err := ref.sb.Pause(); err != nil {
		s.releaseSlot(ref)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		// rare Close-during-Pause race: sandbox is paused, unreachable, leaked until pool.Cleanup at process exit
		s.releaseSlotLocked(ref)
		return
	}
	ref.inUse = false
}

// releaseSlotLocked clears sb and inUse. Caller must hold s.mu.
func (s *sandboxSetImpl) releaseSlotLocked(ref *SandboxRef) {
	ref.sb = nil
	ref.inUse = false
}

func (s *sandboxSetImpl) releaseSlot(ref *SandboxRef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !ref.inUse {
		panic(fmt.Sprintf("sandboxset: releaseSlot on ref %p not currently held (inUse=%v)", ref, ref.inUse))
	}
	s.releaseSlotLocked(ref)
}

// Close clears idle slots; in-use refs are left to their holders, whose put()
// will see s.closed and release them. Never touches a held ref's sb.
// Best-effort: if a holder never calls Put, the slot is not reclaimed.
func (s *sandboxSetImpl) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("Close: already %w", ErrClosed)
	}
	s.closed = true

	for _, ref := range s.pool {
		if !ref.inUse {
			s.releaseSlotLocked(ref)
		}
	}
	s.pool = nil
	return nil
}

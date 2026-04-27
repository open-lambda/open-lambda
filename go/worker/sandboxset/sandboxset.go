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

type SandboxRef struct {
	set   *sandboxSetImpl
	sb    sandbox.Sandbox
	inUse bool
}

// Sandbox returns the underlying sandbox. No inUse guard: hot path; misuse is caught by Put/MarkDead guards instead.
func (r *SandboxRef) Sandbox() sandbox.Sandbox { return r.sb }

func (r *SandboxRef) MarkDead() {
	if !r.inUse {
		panic(fmt.Sprintf("sandboxset: MarkDead on ref %p not currently held (inUse=%v)", r, r.inUse))
	}
	r.sb = nil
}

func (r *SandboxRef) Put() { r.set.put(r) }

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
			s.mu.Lock()
			ref.inUse = false
			s.mu.Unlock()
			return nil, err
		}
		ref.sb = newSb
	}

	return ref, nil
}

func (s *sandboxSetImpl) put(ref *SandboxRef) {
	if ref.sb != nil {
		if err := ref.sb.Pause(); err != nil {
			ref.sb.Destroy("sandboxset: pause failed")
			ref.sb = nil
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if !ref.inUse {
		panic(fmt.Sprintf("sandboxset: put on ref %p not currently held (inUse=%v)", ref, ref.inUse))
	}
	if s.closed {
		sb := ref.sb
		ref.sb = nil
		ref.inUse = false
		if sb != nil {
			sb.Destroy("sandboxset: closed during put")
		}
		return
	}
	ref.inUse = false
}


func (s *sandboxSetImpl) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("Close: already %w", ErrClosed)
	}
	s.closed = true

	for _, ref := range s.pool {
		if !ref.inUse && ref.sb != nil {
			ref.sb.Destroy("sandboxset: closed")
			ref.sb = nil
		}
	}
	return nil
}

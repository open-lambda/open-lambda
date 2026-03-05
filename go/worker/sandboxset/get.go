package sandboxset

import (
	"fmt"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// Get implements SandboxSet.
//
// Fast path: an idle sandbox is claimed under a short lock, then Unpause
// runs outside the lock so the pool is not stalled during I/O.
//
// Slow path: no idle sandbox exists, so a new one is created without
// holding the lock.
func (s *sandboxSetImpl) Get() (sandbox.Sandbox, error) {
	// Loop over idle sandboxes until one unpauses successfully,
	// or the pool has no idle sandboxes left.
	for {
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			return nil, fmt.Errorf("sandboxset: closed")
		}

		// Fast path: claim an idle sandbox.
		var claimed sandbox.Sandbox
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
		return claimed, nil
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

	return sb, nil
}

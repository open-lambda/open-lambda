package sandboxset

import (
	"fmt"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

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

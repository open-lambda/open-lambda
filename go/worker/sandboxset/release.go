package sandboxset

import (
	"fmt"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// ReleaseSandbox implements SandboxSet.
//
// The sandbox is paused (unless WithoutPause was supplied) and its wrapper is
// flipped back to idle. If Pause fails the sandbox is destroyed rather than
// silently recycled — a bad sandbox should never re-enter the pool.
func (s *sandboxSetImpl) ReleaseSandbox(sb sandbox.Sandbox, opts ...ReleaseOption) error {
	o := &releaseOptions{}
	for _, opt := range opts {
		opt(o)
	}

	if !o.skipPause {
		if err := sb.Pause(); err != nil {
			_ = s.DestroyAndRemove(sb, fmt.Sprintf("pause failed on release: %v", err))
			return fmt.Errorf("sandboxset: sandbox %s destroyed because Pause failed: %w", sb.ID(), err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, w := range s.pool {
		if w.sb.ID() == sb.ID() {
			w.inUse = false
			s.metrics.Releases++
			return nil
		}
	}
	return fmt.Errorf("sandboxset: sandbox %s not found in pool", sb.ID())
}

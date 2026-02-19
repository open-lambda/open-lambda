package sandboxset

import "fmt"

// Warm implements SandboxSet.
//
// Sandboxes are created and paused sequentially. Parallel creation could
// overwhelm the host with container-start overhead; pools are typically small
// (5–10 entries) so sequential creation is fast enough.
func (s *sandboxSetImpl) Warm(target int) error {
	for {
		s.mu.RLock()
		current := len(s.pool)
		s.mu.RUnlock()

		if current >= target {
			return nil
		}
		if s.cfg.MaxSize > 0 && current >= s.cfg.MaxSize {
			return nil
		}

		sb, err := s.cfg.Pool.Create(
			s.cfg.Parent, s.cfg.IsLeaf,
			s.cfg.CodeDir, s.cfg.ScratchDir,
			s.cfg.Meta,
		)
		if err != nil {
			return fmt.Errorf("sandboxset: Warm create[%d]: %w", current, err)
		}

		if err := sb.Pause(); err != nil {
			sb.Destroy("pause failed during Warm")
			return fmt.Errorf("sandboxset: Warm pause[%d]: %w", current, err)
		}

		s.mu.Lock()
		s.pool = append(s.pool, &sandboxWrapper{sb: sb, inUse: false})
		s.mu.Unlock()
	}
}

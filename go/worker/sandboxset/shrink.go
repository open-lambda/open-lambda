package sandboxset

// Shrink implements SandboxSet.
//
// One idle sandbox is removed and destroyed per iteration so the write lock
// is never held while destruction I/O runs.
func (s *sandboxSetImpl) Shrink(target int) error {
	for {
		s.mu.Lock()
		if len(s.pool) <= target {
			s.mu.Unlock()
			return nil
		}

		var victim *sandboxWrapper
		for i, w := range s.pool {
			if !w.inUse {
				victim = w
				s.pool[i] = s.pool[len(s.pool)-1]
				s.pool = s.pool[:len(s.pool)-1]
				break
			}
		}
		s.mu.Unlock()

		if victim == nil {
			// All remaining sandboxes are actively in use.
			return nil
		}
		victim.sb.Destroy("shrink")
	}
}

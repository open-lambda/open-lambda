package sandboxset

import "fmt"

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

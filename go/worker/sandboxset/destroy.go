package sandboxset

import (
	"fmt"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// DestroyAndRemove implements SandboxSet.
//
// The wrapper is spliced out of the pool under a short write lock using O(1)
// swap-with-tail. Destroy is called outside the lock to keep critical sections
// short. The sandbox is always destroyed even if it was not found in the pool.
func (s *sandboxSetImpl) DestroyAndRemove(sb sandbox.Sandbox, reason string) error {
	s.mu.Lock()
	found := false
	for i, w := range s.pool {
		if w.sb.ID() == sb.ID() {
			s.pool[i] = s.pool[len(s.pool)-1]
			s.pool = s.pool[:len(s.pool)-1]
			s.metrics.Destroys++
			found = true
			break
		}
	}
	s.mu.Unlock()

	sb.Destroy(reason)

	if !found {
		return fmt.Errorf("sandboxset: sandbox %s not found in pool (still destroyed)", sb.ID())
	}
	return nil
}

package sandboxset

import (
	"fmt"
	"time"

	"github.com/open-lambda/open-lambda/go/worker/sandbox"
)

// GetSandbox implements SandboxSet.
//
// Fast path: an idle sandbox is claimed under a short write lock, then
// Unpause runs outside the lock so the pool is not stalled during I/O.
//
// Slow path: no idle sandbox exists. If the pool has room, a new sandbox is
// created without holding any lock, then appended. If the pool is at MaxSize
// and nothing becomes available before the deadline, an error is returned.
func (s *sandboxSetImpl) GetSandbox(opts ...GetOption) (sandbox.Sandbox, error) {
	o := &getOptions{timeout: s.cfg.DefaultTimeout}
	for _, opt := range opts {
		opt(o)
	}

	var deadline time.Time
	if o.timeout > 0 {
		deadline = time.Now().Add(o.timeout)
	}

	s.mu.Lock()
	s.metrics.Gets++
	s.mu.Unlock()

	for {
		// Try to claim an idle sandbox.
		s.mu.Lock()
		var claimed sandbox.Sandbox
		for _, w := range s.pool {
			if !w.inUse {
				w.inUse = true
				claimed = w.sb
				break
			}
		}
		atCap := s.cfg.MaxSize > 0 && len(s.pool) >= s.cfg.MaxSize
		poolSize := len(s.pool)
		s.mu.Unlock()

		if claimed != nil {
			// Unpause outside the lock (split-lock pattern): inUse=true already
			// prevents another goroutine from claiming this sandbox.
			if err := claimed.Unpause(); err != nil {
				_ = s.DestroyAndRemove(claimed, fmt.Sprintf("unpause: %v", err))
				continue
			}
			s.mu.Lock()
			s.metrics.Hits++
			s.mu.Unlock()
			return claimed, nil
		}

		// Pool is at capacity: wait or timeout.
		if atCap {
			if !deadline.IsZero() && time.Now().After(deadline) {
				s.mu.Lock()
				s.metrics.Timeouts++
				s.mu.Unlock()
				return nil, fmt.Errorf(
					"sandboxset: all %d sandboxes in use; deadline exceeded", poolSize,
				)
			}
			time.Sleep(1 * time.Millisecond)
			continue
		}

		// Slow path: create a new sandbox without holding the lock.
		sb, err := s.cfg.Pool.Create(
			s.cfg.Parent, s.cfg.IsLeaf,
			s.cfg.CodeDir, s.cfg.ScratchDir,
			s.cfg.Meta,
		)
		if err != nil {
			return nil, fmt.Errorf("sandboxset: create sandbox: %w", err)
		}

		s.mu.Lock()
		s.pool = append(s.pool, &sandboxWrapper{sb: sb, inUse: true})
		s.metrics.Misses++
		s.mu.Unlock()

		return sb, nil
	}
}

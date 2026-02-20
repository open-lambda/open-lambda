package sandboxset

// Metrics implements SandboxSet. Returns a copy so callers cannot mutate
// internal counters.
func (s *sandboxSetImpl) Metrics() *Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m := s.metrics
	return &m
}

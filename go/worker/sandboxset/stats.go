package sandboxset

// Stats implements SandboxSet.
func (s *sandboxSetImpl) Stats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	available, inUse := 0, 0
	for _, w := range s.pool {
		if w.inUse {
			inUse++
		} else {
			available++
		}
	}
	return map[string]int{
		"available": available,
		"in_use":    inUse,
		"total":     len(s.pool),
	}
}

package cache

type SubsetMatcher struct {
}

func NewSubsetMatcher() *SubsetMatcher {
	return &SubsetMatcher{}
}

func (sm *SubsetMatcher) Match(servers []*ForkServer, imports []string) (*ForkServer, []string, bool) {
	best_fs := servers[0]
	best_score := -1
	best_toCache := imports
	for i := 1; i < len(servers); i++ {
		matched := 0
		toCache := make([]string, 0, 0)
		for j := 0; j < len(imports); j++ {
			if servers[i].Imports[imports[j]] {
				matched += 1
			} else {
				toCache = append(toCache, imports[j])
			}
		}

		// constrain to subset
		if matched > best_score && len(servers[i].Imports) <= matched {
			best_fs = servers[i]
			best_score = matched
			best_toCache = toCache
		}
	}

	return best_fs, best_toCache, best_score != -1
}

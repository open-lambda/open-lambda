package policy

type SubsetMatcher struct {
}

func NewSubsetMatcher() *SubsetMatcher {
	return &SubsetMatcher{}
}

func (sm *SubsetMatcher) Match(servers []*ForkServer, pkgs []string) (*ForkServer, []string) {
	best_fs := servers[0]
	best_score := 0
	best_toCache := pkgs
	for i := 1; i < len(servers); i++ {
		matched := 0
		toCache := make([]string, 0, 0)
		for j := 0; j < len(pkgs); j++ {
			if servers[i].Packages[pkgs[j]] {
				matched += 1
			} else {
				toCache = append(toCache, pkgs[j])
			}
		}

		// constrain to subset
		if matched > best_score && len(servers[i].Packages) <= matched {
			best_fs = servers[i]
			best_score = matched
			best_toCache = toCache
		}
	}

	return best_fs, best_toCache
}

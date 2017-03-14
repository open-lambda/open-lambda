package policy

import (
	"math/rand"
	"time"
)

type SubsetMatcher struct {
	servers []ForkServer
}

func NewSubsetMatcher(servers []ForkServer) *SubsetMatcher {
	rand.Seed(time.Now().Unix())

	return &SubsetMatcher{
		servers: servers,
	}
}

func (sm *SubsetMatcher) Match(req_pkgs []string) (*ForkServer, []string) {
    best := sm.servers[0]
    best_score := 0
    for i := 0; i < len(sm.servers); i++ {
        matched := 0
        for j := 0; j < len(req_pkgs); j++ {
            if sm.servers[i].Packages[req_pkgs[j]] {
                matched += 1
            }
        }

        // constrain to subset
        if matched > best_score && len(sm.servers[i].Packages) <= matched {
            best = sm.servers[i]
            best_score = matched
        }
    }

	return &best, req_pkgs
}

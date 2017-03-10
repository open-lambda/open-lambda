package policy

import (
	"math/rand"
	"time"
)

type RandomMatcher struct {
	servers []ForkServer
}

func NewRandomMatcher(servers []ForkServer) *RandomMatcher {
	rand.Seed(time.Now().Unix())

	return &RandomMatcher{
		servers: servers,
	}
}

func (rm *RandomMatcher) Match(req_pkgs []string) (*ForkServer, []string) {
	k := rand.Int() % len(rm.servers)

	return &rm.servers[k], req_pkgs
}

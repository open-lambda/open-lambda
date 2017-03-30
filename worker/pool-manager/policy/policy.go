package policy

import (
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type ForkServer struct {
	Sandbox  sb.ContainerSandbox
	Pid      string
	SockPath string
	Packages map[string]bool
}

type CacheMatcher interface {
	Match(servers []ForkServer, pkgs []string) (*ForkServer, []string)
}

type CacheEvictor interface {
	Evict(servers []ForkServer) error
}

package policy

type ForkServer struct {
	Pid      string
	SockPath string
	Packages map[string]bool
}

type CacheMatcher interface {
	Match(req_pkgs []string) (*ForkServer, []string)
}

type CacheEvictor interface {
	Evict() error
}

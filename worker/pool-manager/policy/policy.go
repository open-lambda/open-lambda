package policy

type ForkServer struct {
	Pid      string
	SockPath string
	Packages []string
}

type CacheMatcher interface {
	Match(req_pkgs []string) (*ForkServer, []string)
}

type CacheEvictor interface {
	Evict() error
}

package policy

type ForkServer struct {
	Pid      string
	SockPath string
	Packages []string
}

type CacheMatcher interface {
	Match(request_pkgs []string) (*ForkServer, []string)
}

type CacheEvictor interface {
	Evict() error
}

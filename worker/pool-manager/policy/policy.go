package policy

type CacheMatcher interface {
	Match(servers []*ForkServer, pkgs []string) (*ForkServer, []string)
}

type CacheEvictor interface {
	Evict(servers []*ForkServer) error
}

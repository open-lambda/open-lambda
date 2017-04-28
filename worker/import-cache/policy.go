package cache

type CacheMatcher interface {
	Match(servers []*ForkServer, pkgs []string) (*ForkServer, []string, bool)
}

type CacheEvictor interface {
	Evict(servers []*ForkServer) error
}

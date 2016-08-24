package olreg

import (
	r "github.com/open-lambda/code-registry/registry"
)

const (
	CHUNK_SIZE = 1024
	DATABASE   = "olregistry"
	HANDLER    = "handler"
	SPORT      = 10000
	TABLE      = "handlers"
)

type PushClient struct {
	Client *r.PushClient
}

type PullClient struct {
	Client *r.PullClient
}

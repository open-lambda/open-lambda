package pmanager

import (
	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/pool-manager/policy"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type PoolManager interface {
	Provision(sandbox sb.ContainerSandbox, dir string, pkgs []string) (*policy.ForkServer, bool, error)
}

func InitPoolManager(opts *config.Config) (pm PoolManager, err error) {
	if opts.Import_cache_size != 0 {
		if pm, err = NewBasicManager(opts); err != nil {
			return nil, err
		}
	} else {
		pm = nil
	}

	return pm, nil
}

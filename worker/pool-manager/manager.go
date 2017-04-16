package pmanager

import (
	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/pool-manager/policy"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type PoolManager interface {
	Provision(sandbox sb.ContainerSandbox, dir string, pkgs []string) (*policy.ForkServer, error)
}

func InitPoolManager(config *config.Config) (pm PoolManager, err error) {
	if config.Pool == "basic" {
		if pm, err = NewBasicManager(config); err != nil {
			return nil, err
		}
	} else {
		pm = nil
	}

	return pm, nil
}

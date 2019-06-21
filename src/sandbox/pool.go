package sandbox

import (
	"fmt"

	"github.com/open-lambda/open-lambda/ol/config"
)

func SandboxPoolFromConfig() (cf SandboxPool, err error) {
	if config.Conf.Sandbox == "docker" {
		return NewDockerPool("", nil, false)
	} else if config.Conf.Sandbox == "sock" {
		handlerSandboxes, err := NewSOCKPool("sock-handlers")
		if err != nil {
			return nil, err
		}

		if config.Conf.Import_cache_mb == 0 {
			return handlerSandboxes, nil
		} else {
			cacheSandboxes, err := NewSOCKPool("sock-cache")
			if err != nil {
				return nil, err
			}

			return NewImportCacheContainerFactory(handlerSandboxes, cacheSandboxes)
		}
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", config.Conf.Sandbox)
}

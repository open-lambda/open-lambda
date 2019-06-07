package sandbox

import (
	"fmt"
	"path/filepath"

	"github.com/open-lambda/open-lambda/ol/config"
)

// ContainerFactory is the common interface for creating containers.
type ContainerFactory interface {
	Create(handlerDir, workingDir string, imports []string) (Sandbox, error)
	Cleanup()
}

func InitHandlerContainerFactory() (cf ContainerFactory, err error) {
	if config.Conf.Sandbox == "docker" {
		return NewDockerContainerFactory("", nil, false)
	} else if config.Conf.Sandbox == "sock" {
		handlerRoots := filepath.Join(config.Conf.Worker_dir, "sock-handler-roots")
		handlerSandboxes, err := NewSOCKContainerFactory(handlerRoots, false)
		if err != nil {
			return nil, err
		}

		if config.Conf.Import_cache_mb == 0 {
			return handlerSandboxes, nil
		} else {
			cacheRoots := filepath.Join(config.Conf.Worker_dir, "sock-cache-roots")
			cacheSandboxes, err := NewSOCKContainerFactory(cacheRoots, true)
			if err != nil {
				return nil, err
			}

			return NewImportCacheContainerFactory(handlerSandboxes, cacheSandboxes)
		}
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", config.Conf.Sandbox)
}

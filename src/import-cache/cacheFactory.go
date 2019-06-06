package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-lambda/open-lambda/ol/config"

	sb "github.com/open-lambda/open-lambda/ol/sandbox"
)

type CacheFactory interface {
	Create() (sb.Sandbox, error)
	Cleanup()
}

type cacheFactory struct {
	delegate sb.ContainerFactory
	cacheDir string
}

func NewCacheFactory() (CacheFactory, sb.Sandbox, string, error) {
	cacheDir := filepath.Join(config.Conf.Worker_dir, "import-cache")
	if err := os.MkdirAll(cacheDir, os.ModeDir); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create pool directory at %s :: %v", cacheDir, err)
	}

	delegate, err := sb.InitCacheContainerFactory()
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to initialize cache sandbox factory :: %v", err)
	}

	factory := &cacheFactory{delegate, cacheDir}

	root, err := factory.Create()
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create root cache entry :: %v", err)
	}

	if err := root.RunServer(); err != nil {
		return nil, nil, "", fmt.Errorf("failed to start server in root cache entry :: %v", err)
	}

	rootEntryDir := filepath.Join(cacheDir, "0")

	return factory, root, rootEntryDir, nil
}

func (cf *cacheFactory) Create() (sb.Sandbox, error) {
	container, err := cf.delegate.Create("", cf.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache entry sandbox :: %v", err)
	}

	if err := container.Start(); err != nil {
		go container.Destroy() // TODO: cleanup in Start if we fail
		return nil, err
	}

	return container, nil
}

func (cf *cacheFactory) Cleanup() {
	cf.delegate.Cleanup()
}

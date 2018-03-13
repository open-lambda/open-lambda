package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-lambda/open-lambda/worker/config"

	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type CacheFactory interface {
	Create() (sb.Container, error)
	Cleanup()
}

type cacheFactory struct {
	delegate sb.ContainerFactory
	cacheDir string
}

func NewCacheFactory(opts *config.Config) (CacheFactory, sb.Container, string, error) {
	cacheDir := filepath.Join(opts.Worker_dir, "import-cache")
	if err := os.MkdirAll(cacheDir, os.ModeDir); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create pool directory at %s :: %v", cacheDir, err)
	}

	delegate, err := sb.InitCacheContainerFactory(opts)
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

func (cf *cacheFactory) Create() (sb.Container, error) {
	container, err := cf.delegate.Create("", cf.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache entry sandbox :: %v", err)
	}

	if err := container.Start(); err != nil {
		go func() {
			container.Stop()
			container.Remove()
		}()
		return nil, err
	}

	return container, nil
}

func (cf *cacheFactory) Cleanup() {
	cf.delegate.Cleanup()
}

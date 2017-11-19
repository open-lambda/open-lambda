package cache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-lambda/open-lambda/worker/config"

	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type CacheFactory interface {
	Create() (sb.ContainerSandbox, error)
	Cleanup()
}

type cacheFactory struct {
	delegate sb.SandboxFactory
	cacheDir string
}

func NewCacheFactory(opts *config.Config) (CacheFactory, sb.ContainerSandbox, string, error) {
	cacheDir := opts.Import_cache_dir
	if err := os.MkdirAll(opts.Import_cache_dir, os.ModeDir); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create pool directory at %s :: %v", cacheDir, err)
	}

	delegate, err := sb.InitCacheSandboxFactory(opts)
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

func (cf *cacheFactory) Create() (sb.ContainerSandbox, error) {
	sandbox, err := cf.delegate.Create("", cf.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache entry sandbox :: %v", err)
	}

	container, ok := sandbox.(sb.ContainerSandbox)
	if !ok {
		return nil, fmt.Errorf("cache only supports container sandboxes")
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

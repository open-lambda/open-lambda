package pmanager

import (
	"fmt"
	"os"
	"path/filepath"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/dockerutil"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

func InitCacheFactory(poolDir, cluster string, buffer int) (cf *BufferedCacheFactory, root *sb.DockerSandbox, rootDir string, err error) {
	cf, root, rootDir, err = NewBufferedCacheFactory(poolDir, cluster, buffer)
	if err != nil {
		return nil, nil, "", err
	}

	return cf, root, rootDir, nil
}

// CacheFactory is a SandboxFactory that creates docker sandboxes for the cache.
type CacheFactory struct {
	client *docker.Client
	cmd    []string
	caps   []string
	labels map[string]string
}

// emptySBInfo wraps sandbox information necessary for the buffer.
type emptySBInfo struct {
	sandbox    *sb.DockerSandbox
	sandboxDir string
}

// BufferedCacheFactory maintains a buffer of sandboxes created by another factory.
type BufferedCacheFactory struct {
	delegate *CacheFactory
	buffer   chan *emptySBInfo
	errors   chan error
	dir      string
}

// NewCacheFactory creates a CacheFactory.
func NewCacheFactory(cluster string) (*CacheFactory, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	cmd := []string{"/init"}

	caps := []string{"SYS_ADMIN"}

	labels := map[string]string{
		dockerutil.DOCKER_LABEL_CLUSTER: cluster,
		dockerutil.DOCKER_LABEL_TYPE:    dockerutil.POOL,
	}

	cf := &CacheFactory{client, cmd, caps, labels}
	return cf, nil
}

// Create creates a docker sandbox from the pool directory.
func (cf *CacheFactory) Create(sandboxDir string, cmd []string) (*sb.DockerSandbox, error) {
	volumes := []string{
		fmt.Sprintf("%s:%s", sandboxDir, "/host"),
	}

	container, err := cf.client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:  dockerutil.CACHE_IMAGE,
				Labels: cf.labels,
				Cmd:    cmd,
			},
			HostConfig: &docker.HostConfig{
				Binds:   volumes,
				PidMode: "host",
				CapAdd:  cf.caps,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	sandbox := sb.NewDockerSandbox(sandboxDir, "", container, cf.client)
	return sandbox, nil
}

// NewBufferedCacheFactory creates a BufferedCacheFactory and starts a go routine to
// fill the sandbox buffer.
func NewBufferedCacheFactory(poolDir, cluster string, buffer int) (*BufferedCacheFactory, *sb.DockerSandbox, string, error) {
	delegate, err := NewCacheFactory(cluster)
	if err != nil {
		return nil, nil, "", err
	}

	bf := &BufferedCacheFactory{
		delegate: delegate,
		buffer:   make(chan *emptySBInfo, buffer),
		errors:   make(chan error, buffer),
		dir:      poolDir,
	}

	if err := os.MkdirAll(poolDir, os.ModeDir); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create pool directory at %s: %v", poolDir, err)
	}

	// create the root container
	rootDir := filepath.Join(bf.dir, "root")
	if err := os.MkdirAll(rootDir, os.ModeDir); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create cache entry directory at %s: %v", poolDir, err)
	}

	root, err := bf.delegate.Create(rootDir, []string{"python", "initroot.py"})
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create cache entry sandbox: %v", err)
	} else if err := root.Start(); err != nil {
		return nil, nil, "", fmt.Errorf("failed to start cache entry sandbox: %v", err)
	}

	// fill the sandbox buffer
	go func() {
		idx := 1
		for {
			sandboxDir := filepath.Join(bf.dir, fmt.Sprintf("%d", idx))
			if err := os.MkdirAll(sandboxDir, os.ModeDir); err != nil {
				bf.buffer <- nil
				bf.errors <- err
			} else if sandbox, err := bf.delegate.Create(sandboxDir, []string{"/init"}); err != nil {
				bf.buffer <- nil
				bf.errors <- err
			} else if err := sandbox.Start(); err != nil {
				bf.buffer <- nil
				bf.errors <- err
			} else if err := sandbox.Pause(); err != nil {
				bf.buffer <- nil
				bf.errors <- err
			} else {
				bf.buffer <- &emptySBInfo{sandbox, sandboxDir}
				bf.errors <- nil
			}
			idx++
		}
	}()

	return bf, root, rootDir, nil
}

// Returns a sandbox ready for a cache interpreter
func (bf *BufferedCacheFactory) Create() (*sb.DockerSandbox, string, error) {
	info, err := <-bf.buffer, <-bf.errors
	if err != nil {
		return nil, "", err
	}

	if err := info.sandbox.Unpause(); err != nil {
		return nil, "", err
	}

	return info.sandbox, info.sandboxDir, nil
}

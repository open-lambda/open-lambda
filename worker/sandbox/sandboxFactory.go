package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/dockerutil"
)

// SandboxFactory is the common interface for all sandbox creation functions.
type SandboxFactory interface {
	Create(handlerDir string, sandboxDir string) (sandbox Sandbox, err error)
}

func InitSandboxFactory(config *config.Config) (sf SandboxFactory, err error) {
	if df, err := NewDockerSBFactory(config); err != nil {
		return nil, err
	} else if config.Sandbox_buffer == 0 {
		return df, nil
	} else {
		return NewBufferedSBFactory(config, df)
	}
}

// DockerSBFactory is a SandboxFactory that creats docker sandboxes.
type DockerSBFactory struct {
	client *docker.Client
	cmd    []string
	labels map[string]string
	env    []string
}

// emptySBInfo wraps sandbox information necessary for the buffer.
type emptySBInfo struct {
	sandbox    Sandbox
	handlerDir string
	sandboxDir string
}

// BufferedSBFactory maintains a buffer of sandboxes created by another factory.
type BufferedSBFactory struct {
	delegate SandboxFactory
	buffer   chan *emptySBInfo
	errors   chan error
	mntDir   string
}

// NewDockerSBFactory creates a DockerSBFactory.
func NewDockerSBFactory(opts *config.Config) (*DockerSBFactory, error) {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	labels := map[string]string{
		dockerutil.DOCKER_LABEL_CLUSTER: opts.Cluster_name,
		dockerutil.DOCKER_LABEL_TYPE:    dockerutil.SANDBOX,
	}
	env := []string{fmt.Sprintf("ol.config=%s", opts.SandboxConfJson())}
	var cmd []string
	if opts.Pool == "" {
		cmd = []string{"/usr/bin/python", "/server.py"}
	} else {
		cmd = []string{"/init"}
	}

	df := &DockerSBFactory{c, cmd, labels, env}
	return df, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (df *DockerSBFactory) Create(handlerDir string, sandboxDir string) (Sandbox, error) {
	volumes := []string{
		fmt.Sprintf("%s:%s:ro,slave", handlerDir, "/handler"),
		fmt.Sprintf("%s:%s:slave", sandboxDir, "/host"),
	}
	container, err := df.client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:  dockerutil.BASE_IMAGE,
				Labels: df.labels,
				Env:    df.env,
				Cmd:    df.cmd,
			},
			// MM: Add Privileged flag to allow for perf
			HostConfig: &docker.HostConfig{
				Privileged: true,
				Binds:      volumes,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	sandbox := NewDockerSandbox(sandboxDir, container, df.client)
	return sandbox, nil
}

// mkSBDirs makes the handler and sandbox directories and tries to unmount them.
func mkSBDirs(bufDir string) (string, string, error) {
	if err := os.MkdirAll(bufDir, os.ModeDir); err != nil {
		return "", "", fmt.Errorf("fail to create directory at %s: %v", bufDir, err)
	}
	handlerDir := filepath.Join(bufDir, "handler")
	if err := os.MkdirAll(handlerDir, os.ModeDir); err != nil {
		return "", "", fmt.Errorf("fail to create directory at %s: %v", handlerDir, err)
	}
	if err := syscall.Unmount(handlerDir, 0); err != nil && err != syscall.EINVAL {
		return "", "", fmt.Errorf("fail to unmount directory %s: %v", handlerDir, err)
	}
	sandboxDir := filepath.Join(bufDir, "host")
	if err := os.MkdirAll(sandboxDir, os.ModeDir); err != nil {
		return "", "", fmt.Errorf("fail to create directory at %s: %v", sandboxDir, err)
	}
	if err := syscall.Unmount(sandboxDir, 0); err != nil && err != syscall.EINVAL {
		return "", "", fmt.Errorf("fail to unmount directory %s: %v", sandboxDir, err)
	}
	return handlerDir, sandboxDir, nil
}

// NewBufferedSBFactory creates a BufferedSBFactory and starts a go routine to
// fill the sandbox buffer.
func NewBufferedSBFactory(opts *config.Config, delegate SandboxFactory) (*BufferedSBFactory, error) {
	bf := &BufferedSBFactory{}
	bf.delegate = delegate
	bf.buffer = make(chan *emptySBInfo, opts.Sandbox_buffer-1) // -1 for the last one blocking the channel
	bf.errors = make(chan error, opts.Sandbox_buffer-1)
	bf.mntDir = "/tmp/.olmnts"

	if err := os.MkdirAll(bf.mntDir, os.ModeDir); err != nil {
		return nil, fmt.Errorf("fail to create directory at %s: %v", bf.mntDir, err)
	}

	// fill the sandbox buffer
	go func() {
		idx := 0
		for {
			bufDir := filepath.Join(bf.mntDir, fmt.Sprintf("%d", idx))
			if handlerDir, sandboxDir, err := mkSBDirs(bufDir); err != nil {
				bf.buffer <- nil
				bf.errors <- err
			} else if sandbox, err := bf.delegate.Create(handlerDir, sandboxDir); err != nil {
				bf.buffer <- nil
				bf.errors <- err
			} else if err := sandbox.Start(); err != nil {
				bf.buffer <- nil
				bf.errors <- err
			} else if err := sandbox.Pause(); err != nil {
				bf.buffer <- nil
				bf.errors <- err
			} else {
				bf.buffer <- &emptySBInfo{sandbox, handlerDir, sandboxDir}
				bf.errors <- nil
			}
			idx++
		}
	}()

	return bf, nil
}

// Create mounts the handler and sandbox directories to the ones already
// mounted in the sandbox, and returns that sandbox. The sandbox would be in
// Paused state, instead of Stopped.
func (bf *BufferedSBFactory) Create(handlerDir string, sandboxDir string) (Sandbox, error) {
	mntFlag := uintptr(syscall.MS_BIND | syscall.MS_REC)
	if info, err := <-bf.buffer, <-bf.errors; err != nil {
		return nil, err
	} else if err := syscall.Mount(handlerDir, info.handlerDir, "", mntFlag, ""); err != nil {
		return nil, err
	} else if err := syscall.Mount(sandboxDir, info.sandboxDir, "", mntFlag, ""); err != nil {
		return nil, err
	} else {
		return info.sandbox, nil
	}
}

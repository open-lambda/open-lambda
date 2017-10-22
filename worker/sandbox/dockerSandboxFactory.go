package sandbox

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/dockerutil"
)

// DockerSBFactory is a SandboxFactory that creats docker sandboxes.
type DockerSBFactory struct {
	client    *docker.Client
	cmd       []string
	labels    map[string]string
	env       []string
	pkgsDir   string
	indexHost string
	indexPort string
}

// NewDockerSBFactory creates a DockerSBFactory.
func NewDockerSBFactory(opts *config.Config) (*DockerSBFactory, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	labels := map[string]string{
		dockerutil.DOCKER_LABEL_CLUSTER: opts.Cluster_name,
		dockerutil.DOCKER_LABEL_TYPE:    dockerutil.SANDBOX,
	}
	env := []string{fmt.Sprintf("ol.config=%s", opts.SandboxConfJson())}
	cmd := []string{"/ol-init"}

	df := &DockerSBFactory{
		client:    client,
		cmd:       cmd,
		labels:    labels,
		env:       env,
		pkgsDir:   opts.Pkgs_dir,
		indexHost: opts.Index_host,
		indexPort: opts.Index_port,
	}

	return df, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (df *DockerSBFactory) Create(handlerDir, workingDir string) (Sandbox, error) {
	id_bytes, err := exec.Command("uuidgen").Output()
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(id_bytes[:]))

	// create sandbox directory
	hostDir := filepath.Join(workingDir, id)
	if err := os.MkdirAll(hostDir, 0777); err != nil {
		return nil, err
	}
	volumes := []string{
		fmt.Sprintf("%s:%s:ro,slave", handlerDir, "/handler"),
		fmt.Sprintf("%s:%s:slave", hostDir, "/host"),
		fmt.Sprintf("%s:%s:ro", df.pkgsDir, "/packages"),
	}

	container, err := df.client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:  dockerutil.BASE_IMAGE,
				Labels: df.labels,
				Env:    df.env,
				Cmd:    df.cmd,
			},
			HostConfig: &docker.HostConfig{
				Binds: volumes,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	sandbox := NewDockerSandbox(id, hostDir, df.indexHost, df.indexPort, container, df.client)
	return sandbox, nil
}

// TODO
func (df *DockerSBFactory) Cleanup() {
	return
}

// BufferedDockerSBFactory maintains a buffer of sandboxes created by another factory.
type BufferedDockerSBFactory struct {
	delegate SandboxFactory
	buffer   chan *emptySBInfo
	errors   chan error
	mntDir   string
	cache    bool
	idxPtr   *int64
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

// NewBufferedDockerSBFactory creates a BufferedDockerSBFactory and starts a go routine to
// fill the sandbox buffer.
func NewBufferedDockerSBFactory(opts *config.Config, delegate SandboxFactory) (*BufferedDockerSBFactory, error) {
	bf := &BufferedDockerSBFactory{
		delegate: delegate,
		buffer:   make(chan *emptySBInfo, opts.Sandbox_buffer),
		errors:   make(chan error, opts.Sandbox_buffer),
		cache:    opts.Import_cache_size != 0,
	}

	if err := os.MkdirAll(olMntDir, os.ModeDir); err != nil {
		return nil, fmt.Errorf("fail to create directory at %s: %v", olMntDir, err)
	}

	// fill the sandbox buffer
	var sharedIdx int64 = -1
	bf.idxPtr = &sharedIdx

	threads := 1
	if opts.Sandbox_buffer_threads > 0 {
		threads = opts.Sandbox_buffer_threads
	}

	for i := 0; i < threads; i++ {
		go func(idxPtr *int64) {
			for {
				newIdx := atomic.AddInt64(idxPtr, 1)
				if newIdx < 0 {
					return // kill signal
				}

				bufDir := filepath.Join(olMntDir, fmt.Sprintf("%d", newIdx))
				if handlerDir, sandboxDir, err := mkSBDirs(bufDir); err != nil {
					bf.errors <- err
				} else if sandbox, err := bf.delegate.Create(handlerDir, sandboxDir); err != nil {
					bf.errors <- err
				} else if err := sandbox.Start(); err != nil {
					bf.errors <- err
				} else if err := sandbox.Pause(); err != nil {
					bf.errors <- err
				} else {
					bf.buffer <- &emptySBInfo{sandbox, bufDir, handlerDir, sandboxDir}
				}
			}
		}(bf.idxPtr)
	}

	log.Printf("filling sandbox buffer")
	for len(bf.buffer) < cap(bf.buffer) {
		time.Sleep(20 * time.Millisecond)
	}
	log.Printf("sandbox buffer full")

	return bf, nil
}

// Create mounts the handler and sandbox directories to the ones already
// mounted in the sandbox, and returns that sandbox. The sandbox would be in
// Paused state, instead of Stopped.
func (bf *BufferedDockerSBFactory) Create(handlerDir, workingDir string) (Sandbox, error) {
	mntFlag := uintptr(syscall.MS_BIND)
	select {
	case info := <-bf.buffer:
		// create cluster host directory
		hostDir := filepath.Join(workingDir, info.sandbox.ID())
		if err := os.MkdirAll(hostDir, 0777); err != nil {
			return nil, err
		}

		sbHostDir := filepath.Join(info.sandboxDir, info.sandbox.ID())
		if err := info.sandbox.Unpause(); err != nil {
			return nil, err
		} else if err := syscall.Mount(handlerDir, info.handlerDir, "", mntFlag, ""); err != nil {
			return nil, err
		} else if err := syscall.Mount(hostDir, sbHostDir, "", mntFlag, ""); err != nil {
			return nil, err
		}
		if !bf.cache {
			sockPath := filepath.Join(workingDir, "ol.sock")
			_ = os.Remove(sockPath)
		}

		return info.sandbox, nil

	case err := <-bf.errors:
		return nil, err
	}
}

func (bf *BufferedDockerSBFactory) Cleanup() {
	// kill signal must be negative for all producers
	atomic.StoreInt64(bf.idxPtr, -1000)

	// empty the buffer
	start := time.Now()
	for {
		if time.Since(start) > 5*time.Second {
			log.Printf("emptying docker SB buffer took longer than 5s, aborting\n")
			return
		}

		select {
		case info := <-bf.buffer:
			if info == nil {
				continue
			}
			info.sandbox.Unpause()
			info.sandbox.Stop()
			info.sandbox.Remove()

		default:
			bf.delegate.Cleanup()

			syscall.Unmount(olMntDir, syscall.MNT_DETACH)
			os.RemoveAll(olMntDir)

			return
		}
	}
}

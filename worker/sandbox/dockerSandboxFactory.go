package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/open-lambda/open-lambda/worker/config"
)

// DockerSBFactory is a SandboxFactory that creats docker sandboxes.
type DockerSBFactory struct {
	client    *docker.Client
	labels    map[string]string
	caps      []string
	pidMode   string
	image     string
	pkgsDir   string
	indexHost string
	indexPort string
	idxPtr    *int64
}

// NewDockerSBFactory creates a DockerSBFactory.
func NewDockerSBFactory(opts *config.Config, image, pidMode string, caps []string, labels map[string]string) (*DockerSBFactory, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	var sharedIdx int64 = -1
	idxPtr := &sharedIdx

	df := &DockerSBFactory{
		client:    client,
		labels:    labels,
		caps:      caps,
		pidMode:   pidMode,
		image:     image,
		pkgsDir:   opts.Pkgs_dir,
		indexHost: opts.Index_host,
		indexPort: opts.Index_port,
		idxPtr:    idxPtr,
	}

	return df, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (df *DockerSBFactory) Create(handlerDir, workingDir string) (Sandbox, error) {
	id := fmt.Sprintf("%d", atomic.AddInt64(df.idxPtr, 1))
	hostDir := filepath.Join(workingDir, id)
	if err := os.MkdirAll(hostDir, 0777); err != nil {
		return nil, err
	}

	volumes := []string{
		fmt.Sprintf("%s:%s", hostDir, "/host"),
		fmt.Sprintf("%s:%s:ro", df.pkgsDir, "/packages"),
	}

	if handlerDir != "" {
		volumes = append(volumes, fmt.Sprintf("%s:%s:ro", handlerDir, "/handler"))
	}

	// pipe for synchronization before socket is ready
	pipe := filepath.Join(hostDir, "server_pipe")
	if err := syscall.Mkfifo(pipe, 0777); err != nil {
		return nil, err
	}

	container, err := df.client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Cmd:    []string{"/spin"},
				Image:  df.image,
				Labels: df.labels,
			},
			HostConfig: &docker.HostConfig{
				Binds:   volumes,
				CapAdd:  df.caps,
				PidMode: df.pidMode,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	sandbox := NewDockerSandbox(id, hostDir, df.indexHost, df.indexPort, container, df.client)
	return sandbox, nil
}

func (df *DockerSBFactory) Cleanup() {
	return
}

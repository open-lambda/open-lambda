package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/dockerutil"
)

// DockerContainerFactory is a ContainerFactory that creats docker containers.
type DockerContainerFactory struct {
	client         *docker.Client
	labels         map[string]string
	caps           []string
	pidMode        string
	pkgsDir        string
	idxPtr         *int64
	cache          bool
	docker_runtime string
}

// NewDockerContainerFactory creates a DockerContainerFactory.
func NewDockerContainerFactory(opts *config.Config, pidMode string, caps []string, labels map[string]string, cache bool) (*DockerContainerFactory, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	var sharedIdx int64 = -1
	idxPtr := &sharedIdx

	df := &DockerContainerFactory{
		client:  client,
		labels:  labels,
		caps:    caps,
		pidMode: pidMode,
		pkgsDir: opts.Pkgs_dir,
		idxPtr:  idxPtr,
		cache:   cache,
		docker_runtime: opts.Docker_runtime,
	}

	return df, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (df *DockerContainerFactory) Create(handlerDir, workingDir string) (Container, error) {
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
				Image:  dockerutil.LAMBDA_IMAGE,
				Labels: df.labels,
			},
			HostConfig: &docker.HostConfig{
				Binds:   volumes,
				CapAdd:  df.caps,
				PidMode: df.pidMode,
				Runtime: df.docker_runtime,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	sandbox := NewDockerContainer(id, hostDir, df.cache, container, df.client)
	return sandbox, nil
}

func (df *DockerContainerFactory) Cleanup() {
	return
}

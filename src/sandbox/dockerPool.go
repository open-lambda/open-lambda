package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/sandbox/dockerutil"
)

// DockerPool is a ContainerFactory that creats docker containers.
type DockerPool struct {
	client         *docker.Client
	labels         map[string]string
	caps           []string
	pidMode        string
	pkgsDir        string
	idxPtr         *int64
	cache          bool
	docker_runtime string

	sync.Mutex
	sandboxes []Sandbox
}

// NewDockerPool creates a DockerPool.
func NewDockerPool(pidMode string, caps []string, cache bool) (*DockerPool, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	var sharedIdx int64 = -1
	idxPtr := &sharedIdx

	labels := map[string]string{
		dockerutil.DOCKER_LABEL_CLUSTER: config.Conf.Cluster_name,
	}

	pool := &DockerPool{
		client:         client,
		labels:         labels,
		caps:           caps,
		pidMode:        pidMode,
		pkgsDir:        config.Conf.Pkgs_dir,
		idxPtr:         idxPtr,
		cache:          cache,
		docker_runtime: config.Conf.Docker_runtime,
	}

	return pool, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (pool *DockerPool) Create(handlerDir, scratchPrefix string, imports []string) (Sandbox, error) {
	id := fmt.Sprintf("%d", atomic.AddInt64(pool.idxPtr, 1))
	hostDir := filepath.Join(scratchPrefix, id)
	if err := os.MkdirAll(hostDir, 0777); err != nil {
		return nil, err
	}

	volumes := []string{
		fmt.Sprintf("%s:%s", hostDir, "/host"),
		fmt.Sprintf("%s:%s:ro", pool.pkgsDir, "/packages"),
	}

	if handlerDir != "" {
		volumes = append(volumes, fmt.Sprintf("%s:%s:ro", handlerDir, "/handler"))
	}

	// pipe for synchronization before socket is ready
	pipe := filepath.Join(hostDir, "server_pipe")
	if err := syscall.Mkfifo(pipe, 0777); err != nil {
		return nil, err
	}

	container, err := pool.client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Cmd:    []string{"/spin"},
				Image:  dockerutil.LAMBDA_IMAGE,
				Labels: pool.labels,
			},
			HostConfig: &docker.HostConfig{
				Binds:   volumes,
				CapAdd:  pool.caps,
				PidMode: pool.pidMode,
				Runtime: pool.docker_runtime,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	c, err := NewDockerContainer(id, hostDir, pool.cache, container, pool.client)
	if err != nil {
		return nil, err
	}

	// TODO: have some way to clean up this structure as sandboxes are released
	pool.Mutex.Lock()
	pool.sandboxes = append(pool.sandboxes, c)
	pool.Mutex.Unlock()

	return c, nil
}

func (pool *DockerPool) Cleanup() {
	pool.Mutex.Lock()
	for _, sandbox := range pool.sandboxes {
		sandbox.Destroy()
	}
	pool.Mutex.Unlock()
}

func (pool *DockerPool) DebugString() string {
	pool.Mutex.Lock()
	defer pool.Mutex.Unlock()

	var sb strings.Builder

	for _, sandbox := range pool.sandboxes {
		sb.WriteString(fmt.Sprintf("----\n%s\n----\n", sandbox.DebugString()))
	}

	return sb.String()
}

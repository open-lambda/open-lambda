package sandbox

import (
	"fmt"
	"path/filepath"
	"sync/atomic"
	"syscall"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/open-lambda/open-lambda/ol/common"
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
	docker_runtime string
	eventHandlers  []SandboxEventFunc
	debugger
}

// NewDockerPool creates a DockerPool.
func NewDockerPool(pidMode string, caps []string) (*DockerPool, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	var sharedIdx int64 = -1
	idxPtr := &sharedIdx

	labels := map[string]string{
		dockerutil.DOCKER_LABEL_CLUSTER: common.Conf.Worker_dir,
	}

	pool := &DockerPool{
		client:         client,
		labels:         labels,
		caps:           caps,
		pidMode:        pidMode,
		pkgsDir:        common.Conf.Pkgs_dir,
		idxPtr:         idxPtr,
		docker_runtime: common.Conf.Docker_runtime,
		eventHandlers:  []SandboxEventFunc{},
	}

	pool.debugger = newDebugger(pool)

	return pool, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (pool *DockerPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta) (sb Sandbox, err error) {
	meta = fillMetaDefaults(meta)
	t := common.T0("Create()")
	defer t.T1()

	if parent != nil {
		panic("Create parent not supported for DockerPool")
	} else if !isLeaf {
		panic("Non-leaves not supported for DockerPool")
	}

	id := fmt.Sprintf("%d", atomic.AddInt64(pool.idxPtr, 1))

	volumes := []string{
		fmt.Sprintf("%s:%s", scratchDir, "/host"),
		fmt.Sprintf("%s:%s:ro", pool.pkgsDir, "/packages"),
	}

	if codeDir != "" {
		volumes = append(volumes, fmt.Sprintf("%s:%s:ro", codeDir, "/handler"))
	}

	// pipe for synchronization before socket is ready
	pipe := filepath.Join(scratchDir, "server_pipe")
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

	c := &DockerContainer{
		host_id:   id,
		hostDir:   scratchDir,
		container: container,
		client:    pool.client,
		installed: make(map[string]bool),
		meta:      meta,
	}

	if err := c.start(); err != nil {
		c.Destroy()
		return nil, err
	}

	if err := c.runServer(); err != nil {
		c.Destroy()
		return nil, err
	}

	if err := waitForServerPipeReady(c.HostDir()); err != nil {
		c.Destroy()
		return nil, err
	}

	// wrap to make thread-safe and handle container death
	return newSafeSandbox(c, pool.eventHandlers), nil
}

func (pool *DockerPool) Cleanup() {}

func (pool *DockerPool) DebugString() string {
	return pool.debugger.Dump()
}

func (pool *DockerPool) AddListener(handler SandboxEventFunc) {
	pool.eventHandlers = append(pool.eventHandlers, handler)
}

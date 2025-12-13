package sandbox

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/sandbox/dockerutil"
)

// DockerPool is a ContainerFactory that creates docker containers.
type DockerPool struct {
	client        *docker.Client
	labels        map[string]string
	caps          []string
	pidMode       string
	pkgsDir       string
	idxPtr        *int64
	dockerRuntime string
	eventHandlers []SandboxEventFunc
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
		client:        client,
		labels:        labels,
		caps:          caps,
		pidMode:       pidMode,
		pkgsDir:       common.Conf.Pkgs_dir,
		idxPtr:        idxPtr,
		dockerRuntime: common.Conf.Docker.Runtime,
		eventHandlers: []SandboxEventFunc{},
	}

	pool.debugger = newDebugger(pool)

	return pool, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (pool *DockerPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta, _ common.RuntimeType) (sb Sandbox, err error) {
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
	_, statErr := os.Stat(pipe)
	if statErr == nil {
		if err := os.Remove(pipe); err != nil {
			return nil, err
		}
	}
	if statErr == nil || os.IsNotExist(statErr) {
		if err := syscall.Mkfifo(pipe, 0777); err != nil {
			fmt.Println("PIPE CREATION IN Create FAILED")
			return nil, err
		}
	} else {
		return nil, statErr
	}

	// add installed packages to the path
	var pkgDirs []string
	for _, pkg := range meta.Installs {
		pkgDirs = append(pkgDirs, "/packages/"+pkg+"/files")
	}

	// create the container using the specified configuration
	procLimit := int64(common.Conf.Limits.Procs)
	swappiness := int64(common.Conf.Limits.Swappiness)
	cpuPercent := int64(common.Conf.Limits.CPU_percent)
	container, err := pool.client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Cmd:    []string{"/spin"},
				Image:  common.Conf.Docker.Base_image,
				Labels: pool.labels,
				Env:    []string{"PYTHONPATH=" + strings.Join(pkgDirs, ":")},
			},
			HostConfig: &docker.HostConfig{
				Binds:            volumes,
				CapAdd:           pool.caps,
				PidMode:          pool.pidMode,
				Runtime:          pool.dockerRuntime,
				PidsLimit:        &procLimit,
				MemorySwappiness: &swappiness,
				CPUPercent:       cpuPercent,
				Memory:           int64(meta.MemLimitMB * 1024 * 1024),
			},
		},
	)
	if err != nil {
		return nil, err
	}

	c := &DockerContainer{
		hostID:    id,
		hostDir:   scratchDir,
		container: container,
		client:    pool.client,
		installed: make(map[string]bool),
		meta:      meta,
	}

	if err := c.start(); err != nil {
		c.Destroy("c.start() failed")
		return nil, err
	}

	if err := c.runServer(); err != nil {
		c.Destroy("c.runServer() failed")
		return nil, err
	}

	if err := waitForServerPipeReady(c.HostDir()); err != nil {
		c.Destroy("waitForServerPipeReady failed")
		return nil, err
	}

	// start HTTP client
	sockPath := filepath.Join(c.hostDir, "ol.sock")
	if len(sockPath) > 108 {
		return nil, fmt.Errorf("socket path length cannot exceed 108 characters (try moving cluster closer to the root directory")
	}

	dial := func(_, _ string) (net.Conn, error) {
		return net.Dial("unix", sockPath)
	}

	c.httpClient = &http.Client{
		Transport: &http.Transport{Dial: dial},
		Timeout:   time.Second * time.Duration(common.Conf.Limits.Runtime_sec),
	}

	// wrap to make thread-safe and handle container death
	safe := newSafeSandbox(c)
	safe.startNotifyingListeners(pool.eventHandlers)
	return safe, nil
}

// Cleanup will free up any unneeded data/resources
// Currently, this function does nothing and cleanup is handled by the docker daemon
func (_ *DockerPool) Cleanup() {}

// DebugString returns debug information
func (pool *DockerPool) DebugString() string {
	return pool.debugger.Dump()
}

// AddListener allows registering event handlers
func (pool *DockerPool) AddListener(handler SandboxEventFunc) {
	pool.eventHandlers = append(pool.eventHandlers, handler)
}

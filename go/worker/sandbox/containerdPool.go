package sandbox

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/sandbox/containerdutil"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// ContainerdPool is a SandboxPool that creates containerd containers.
type ContainerdPool struct {
	client        *containerd.Client
	namespace     string
	runtime       string
	labels        map[string]string
	idxPtr        *int64
	eventHandlers []SandboxEventFunc
	imageRef      containerd.Image
	cleanupOnce   sync.Once // idempotent cleanup
	debugger
}

// NewContainerdPool creates a ContainerdPool.
func NewContainerdPool() (*ContainerdPool, error) {
	client, err := containerd.New(common.Conf.Containerd.SocketAddress,
		containerd.WithDefaultNamespace(common.Conf.Containerd.Namespace),
		containerd.WithTimeout(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to containerd: %w", err)
	}

	ctx := namespaces.WithNamespace(context.Background(), common.Conf.Containerd.Namespace)

	// ensure that containerd is serving
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	serving, err := client.IsServing(healthCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to check containerd health: %w", err)
	}
	if !serving {
		return nil, fmt.Errorf("containerd is not serving at %s", common.Conf.Containerd.SocketAddress)
	}
	slog.Info("Successfully connected to containerd", "socket_address", common.Conf.Containerd.SocketAddress, "namespace", common.Conf.Containerd.Namespace)

	baseImage := common.Conf.Containerd.Base_image
	if err := containerdutil.EnsureImageExists(ctx, client, baseImage); err != nil {
		return nil, fmt.Errorf("failed to ensure image %q exists: %w", baseImage, err)
	}
	image, err := client.GetImage(ctx, baseImage)
	if err != nil {
		return nil, fmt.Errorf("failed to get image %q: %w", baseImage, err)
	}

	var sharedIdx int64 = -1
	pool := &ContainerdPool{
		client:    client,
		namespace: common.Conf.Containerd.Namespace,
		runtime:   common.Conf.Containerd.Runtime,
		labels: map[string]string{
			containerdutil.CONTAINERD_LABEL_CLUSTER: common.Conf.Worker_dir,
			containerdutil.CONTAINERD_LABEL_RUNTIME: common.Conf.Containerd.Runtime,
		},
		idxPtr:        &sharedIdx,
		imageRef:      image,
		eventHandlers: []SandboxEventFunc{},
	}

	pool.debugger = newDebugger(pool)
	containerdutil.RemoveOrphanedResources(ctx, pool.client, common.Conf.Worker_dir) // clean up containers in case ungraceful ol shutdown

	return pool, nil

}

func (pool *ContainerdPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta) (sb Sandbox, err error) {

	socketPath := filepath.Join(scratchDir, "ol.sock")
	if len(socketPath) > 108 {
		return nil, fmt.Errorf("socket path length cannot exceed 108 characters (try moving cluster closer to the root directory)")
	}

	meta = fillMetaDefaults(meta)
	if parent != nil || !isLeaf {
		return nil, fmt.Errorf("currently ContainerdPool does not support forking from a parent, so parent must be nil and isLeaf must be true")
	}
	id := fmt.Sprintf("ctrd-%d", atomic.AddInt64(pool.idxPtr, 1))

	ctx := namespaces.WithNamespace(context.Background(), pool.namespace)

	mounts := []specs.Mount{
		{
			Destination: "/host",
			Source:      scratchDir,
			Type:        "bind",
			Options:     []string{"rbind", "rw"},
		},
		{
			Destination: "/packages",
			Source:      common.Conf.Pkgs_dir,
			Type:        "bind",
			Options:     []string{"rbind", "ro"},
		},
	}
	if codeDir != "" {
		mounts = append(mounts, specs.Mount{
			Destination: "/handler",
			Source:      codeDir,
			Type:        "bind",
			Options:     []string{"rbind", "ro"},
		})
	}

	pipe := filepath.Join(scratchDir, "server_pipe")
	if _, err := os.Stat(pipe); err == nil {
		if err := os.Remove(pipe); err != nil {
			return nil, fmt.Errorf("failed to remove existing pipe %q: %w", pipe, err)
		}
	} else if !os.IsNotExist(err) {
		// os.Stat failed with unexpected error (not "doesn't exist")
		return nil, fmt.Errorf("failed to check pipe %q: %w", pipe, err)
	}

	if err := syscall.Mkfifo(pipe, 0666); err != nil {
		return nil, fmt.Errorf("failed to create pipe %q: %w", pipe, err)
	}

	var container containerd.Container
	var task containerd.Task
	var execProcess containerd.Process

	defer func() {
		if err != nil {
			containerdutil.CleanupContainerdResources(ctx, "", container, task, execProcess)
		}
	}()

	// Resource limit calculations
	cpuPeriod := uint64(100000)                                                // using Standard CFS period: 100ms (100,000 microseconds)
	cpuQuota := int64(common.Conf.Limits.CPU_percent) * int64(cpuPeriod) / 100 // Convert CPU percentage to microseconds quota
	memoryBytes := uint64(meta.MemLimitMB) * 1024 * 1024                       // Convert MB to bytes
	pidsLimit := int64(common.Conf.Limits.Procs)

	specOpts := []oci.SpecOpts{
		oci.WithProcessArgs("/spin"),
		oci.WithMounts(mounts),
		oci.WithMemoryLimit(memoryBytes),
		oci.WithPidsLimit(pidsLimit),
		oci.WithCPUCFS(cpuQuota, cpuPeriod),
		// network
		// Use host network (not NAT): matches SOCK behavior. Faster (no veth/bridge/iptables),
		// but lambdas can reach host-local services; treat lambda code as trusted.
		oci.WithHostNamespace(specs.NetworkNamespace),
		oci.WithHostHostsFile,
		oci.WithHostResolvconf, // for "/etc/resolv.conf" for package installations
		oci.WithNoNewPrivileges,
	}

	// This only creates metadata; no processes are running yet.
	container, err = pool.client.NewContainer(
		ctx,
		id,
		containerd.WithImage(pool.imageRef),
		containerd.WithNewSnapshot(id, pool.imageRef), // Creates writable COW filesystem layer
		containerd.WithRuntime(pool.runtime, nil),
		containerd.WithContainerLabels(pool.labels),
		containerd.WithNewSpec(specOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create container %q: %w", id, err)
	}

	// a task is the actual running process, while the container above is just metadata
	task, err = container.NewTask(ctx, cio.NullIO)
	if err != nil {
		return nil, fmt.Errorf("failed to create task for container %q: %w", id, err)
	}

	if err := task.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start task for container %q: %w", id, err)
	}

	// Launch the Python lambda server as an exec process (not PID 1).
	cmd := []string{"/usr/bin/python3", "/runtimes/python/server_legacy.py"}

	spec, err := container.Spec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container spec: %w", err)
	}
	env := make([]string, len(spec.Process.Env))
	copy(env, spec.Process.Env)

	// Add PYTHONPATH with all installed package directories
	var pkgDirs []string
	for _, pkg := range meta.Installs {
		pkgDirs = append(pkgDirs, "/packages/"+pkg+"/files")
	}
	if len(pkgDirs) > 0 {
		env = append(env, "PYTHONPATH="+strings.Join(pkgDirs, ":"))
	}

	logPath := filepath.Join(scratchDir, "stdout")
	execProcess, err = task.Exec(ctx,
		"server",
		&specs.Process{
			Args:            cmd,
			Cwd:             "/",
			Env:             env,
			NoNewPrivileges: true, // ignore privilege-granting bits for python lambda server and its child processes
		},
		cio.LogFile(logPath),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create exec process in container %q: %w", id, err)
	}

	if err := execProcess.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start server process in container %q: %w", id, err)
	}

	if err := WaitForServerPipeReady(scratchDir); err != nil {
		return nil, fmt.Errorf("WaitForServerPipeReady failed for container %q: %w", id, err)
	}
	slog.Info("Container server ready", "container_id", id, "socket_path", socketPath)

	// set up HTTP client with unix socket
	dial := func(_ context.Context, network, addr string) (net.Conn, error) {
		return net.Dial("unix", socketPath)
	}
	httpClient := &http.Client{
		Transport: &http.Transport{DialContext: dial},
		Timeout:   time.Second * time.Duration(common.Conf.Limits.Runtime_sec),
	}

	c := &ContainerdContainer{
		container:   container,
		task:        task,
		execProcess: execProcess,
		ctx:         ctx,
		meta:        meta,
		httpClient:  httpClient,
		scratchDir:  scratchDir,
		isPaused:    false,
	}

	safe := newSafeSandbox(c)
	safe.startNotifyingListeners(pool.eventHandlers)
	return safe, nil
}

func (pool *ContainerdPool) AddListener(handler SandboxEventFunc) {
	pool.eventHandlers = append(pool.eventHandlers, handler)
}

func (pool *ContainerdPool) DebugString() string {
	return pool.debugger.Dump()
}

func (pool *ContainerdPool) Cleanup() {
	pool.cleanupOnce.Do(func() {
		if pool.client == nil { // if NewContainerdPool failed partially
			return
		}
		ctx := namespaces.WithNamespace(context.Background(), pool.namespace)
		containerdutil.RemoveOrphanedResources(ctx, pool.client, common.Conf.Worker_dir)
		if err := pool.client.Close(); err != nil {
			slog.Error("Failed to close containerd client", "error", err)
		}
		pool.client = nil
	})

}

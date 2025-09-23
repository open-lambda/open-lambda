package sandbox

import (
        "context"
        "fmt"
        "log/slog"
        "net"
        "net/http"
        "os"
        "path/filepath"
        "sync/atomic"
        "syscall"
        "strings"
        "time"

        "github.com/containerd/containerd"
        "github.com/containerd/containerd/cio"
        "github.com/containerd/containerd/namespaces"
        "github.com/containerd/containerd/oci"
        "github.com/open-lambda/open-lambda/go/common"
        "github.com/open-lambda/open-lambda/go/worker/sandbox/containerdutil"
        "github.com/containerd/containerd/snapshots"
        "github.com/opencontainers/runtime-spec/specs-go"
)

// ContainerdPool is a SandboxPool that creates containerd containers.
type ContainerdPool struct {
        client        *containerd.Client
        namespace     string
        runtime       string
        image         string
        labels        map[string]string
        idxPtr        *int64
        eventHandlers []SandboxEventFunc
        debugger
}

// NewContainerdPool creates a ContainerdPool.
func NewContainerdPool() (*ContainerdPool, error) {
        client, err := containerd.New(common.Conf.Containerd.SocketAddress,
                containerd.WithDefaultNamespace(common.Conf.Containerd.Namespace),
                containerd.WithTimeout(10*time.Second),
        )
        if err != nil {
                return nil, fmt.Errorf("failed to connect to containerd: %v", err)
        }

        ctx := namespaces.WithNamespace(context.Background(), common.Conf.Containerd.Namespace)

        // ensure that containerd is serving
        healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
        defer cancel()
        serving, err := client.IsServing(healthCtx)
        if err != nil {
                return nil, fmt.Errorf("failed to check containerd health: %v", err)
        }
        if !serving {
                return nil, fmt.Errorf("containerd is not serving at %s", common.Conf.Containerd.SocketAddress)
        }
        slog.Info("Successfully connected to containerd", "socket_address", common.Conf.Containerd.SocketAddress, "namespace", common.Conf.Containerd.Namespace)
        var sharedIdx int64 = -1
        pool := &ContainerdPool{
                client:    client,
                namespace: common.Conf.Containerd.Namespace,
                runtime:   common.Conf.Containerd.Runtime,
                image:     common.Conf.Containerd.Base_image,
                labels: map[string]string{
                        containerdutil.CONTAINERD_LABEL_CLUSTER: common.Conf.Worker_dir,
                        containerdutil.CONTAINERD_LABEL_RUNTIME: common.Conf.Containerd.Runtime,
                },
                idxPtr:        &sharedIdx,
                eventHandlers: []SandboxEventFunc{},
        }

        pool.debugger = newDebugger(pool)

        return pool, nil

}


func (pool *ContainerdPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta, rtType common.RuntimeType) (sb Sandbox, err error) {

        socketPath := filepath.Join(scratchDir, "ol.sock")
        if len(socketPath) > 108 { // check ealier to avoid resource cleanup
                return nil, fmt.Errorf("socket path length cannot exceed 108 characters (try moving cluster closer to the root directory")
        }
        
        meta = fillMetaDefaults(meta)
        if parent != nil || !isLeaf {
                return nil, fmt.Errorf("currently ContainerdPool does not support forking from a parent, so parent must be nil and isLeaf must be true")
        }
        id := fmt.Sprintf("ctrd-%d", atomic.AddInt64(pool.idxPtr, 1))

        ctx := namespaces.WithNamespace(context.Background(), pool.namespace)
        if err := containerdutil.EnsureImageExists(ctx, pool.client, pool.image); err != nil {
                return nil, fmt.Errorf("failed to ensure image %q exists: %v", pool.image, err)
        }
        image, err := pool.client.GetImage(ctx, pool.image)
        if err != nil {
                return nil, fmt.Errorf("failed to get image %q: %v", pool.image, err)
        }

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
        _, statErr := os.Stat(pipe)
        if statErr == nil {
                if err := os.Remove(pipe); err != nil {
                        return nil, fmt.Errorf("failed to remove existing pipe %q: %v", pipe, err)
                }
        }
        if statErr == nil || os.IsNotExist(statErr) {
                if err := syscall.Mkfifo(pipe, 0777); err != nil {
                        return nil, fmt.Errorf("failed to create pipe %q: %v", pipe, err)
                }
        } else {
                return nil, statErr
        }

        var (
                container   containerd.Container
                task        containerd.Task
                execProcess containerd.Process
        )

        defer func() {
                if err != nil {
                        // clean up containerd resources associated with this container
                        containerdutil.CleanupContainerdResources(ctx, "", container, task, execProcess)
                }
        }()

        cpuPeriod := uint64(100000) // can be configurable; but for now, period=100ms (standard period)
        container, err = pool.client.NewContainer(
                ctx,
                id,
                containerd.WithImage(image),
                containerd.WithNewSnapshot(id, image),
                containerd.WithRuntime(pool.runtime, nil),
                containerd.WithContainerLabels(pool.labels),
                containerd.WithNewSpec(
                        oci.WithImageConfig(image),
                        oci.WithProcessArgs("/spin"),
                        oci.WithMounts(mounts),
                        oci.WithMemoryLimit(uint64(meta.MemLimitMB)*1024*1024), // should we add python runtime overhead? 
                        oci.WithPidsLimit(int64(common.Conf.Limits.Procs)),
                        oci.WithCPUCFS(int64(common.Conf.Limits.CPU_percent)*int64(cpuPeriod)/100, cpuPeriod),
                        oci.WithNoNewPrivileges,
                ),
        )
        if err != nil {
                return nil, fmt.Errorf("failed to create container %q: %v", id, err)
        }

        task, err = container.NewTask(ctx, cio.NewCreator(cio.WithStdio))
        if err != nil {
                return nil, fmt.Errorf("failed to create task for container %q: %v", id, err)
        }

        if err := task.Start(ctx); err != nil {
                return nil, fmt.Errorf("failed to start task for container %q: %v", id, err)
        }

        // exec python server (like docker's runServer)
        cmd := []string{"python3", "/runtimes/python/server_legacy.py"}

        execProcess, err = task.Exec(ctx,
                "server",
                &specs.Process{
                        Args: cmd,
                        Cwd:  "/",
                        Env: func() []string {
                                // Build PYTHONPATH from installed packages
                                var pkgDirs []string
                                for _, pkg := range meta.Installs {
                                        pkgDirs = append(pkgDirs, "/packages/"+pkg+"/files")
                                }

                                env := []string{ // docker's implementation doesn't set PATH because docker inherits default PATH from base image
                                                "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
                                        }

                                        if len(pkgDirs) > 0 {
                                                env = append(env, "PYTHONPATH="+strings.Join(pkgDirs, ":"))
                                        }

                                        return env
                        }(),
                        NoNewPrivileges: true,
                },
                cio.NewCreator(cio.WithStdio),
        )
        if err != nil {
                return nil, fmt.Errorf("failed to create exec process in container %q: %v", id, err)
        }

        if err := execProcess.Start(ctx); err != nil {
                return nil, fmt.Errorf("failed to start server process in container %q: %v", id, err)
        }

        // wait for server readiness
        if err := WaitForServerPipeReady(scratchDir); err != nil {
                return nil, fmt.Errorf("WaitForServerPipeReady failed for container %q: %v", id, err)
        }
        slog.Info("Container server ready", "container_id", id, "socket_path", socketPath)

        // set up HTTP client with unix socket
        dial := func(_ context.Context, network, addr string) (net.Conn, error) {
                return net.Dial("unix", socketPath) // always use Unix socket
        }
        httpClient := &http.Client{
                Transport: &http.Transport{DialContext: dial},
                Timeout:   time.Second * time.Duration(common.Conf.Limits.Max_runtime_default),
        }

        c := &ContainerdContainer{
                id:          id,
                container:   container,
                task:        task,
                execProcess: execProcess,
                ctx:         ctx,
                client:      pool.client,
                meta:        meta,
                rtType:      rtType,
                httpClient:  httpClient,
                scratchDir:  scratchDir,
                destroyed:   false,
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
        if pool.client == nil { // if NewContainerdPool failed partially
                return
        }

        ctx := namespaces.WithNamespace(context.Background(), pool.namespace)

        // check for any remaining containers with our cluster label
        containers, err := pool.client.Containers(ctx,
                fmt.Sprintf("labels.%q==%q", containerdutil.CONTAINERD_LABEL_CLUSTER, common.Conf.Worker_dir))
        if err != nil {
                slog.Error("Failed to list containers for cleanup", "error", err)
        } else if len(containers) > 0 {
                slog.Warn("Containers remain during ContainerdPool.Cleanup()", "count", len(containers))
                // clean up orphaned containers
                for _, c := range containers {
                        slog.Info("Removing orphaned container", "container_id", c.ID())
                        if err := containerdutil.SafeRemove(ctx, pool.client, c); err != nil {
                                slog.Error("Failed to remove orphaned container", "container_id", c.ID(), "error", err)
                        }
                }
        } else if len(containers) == 0 {
                slog.Info("ContainerdPool cleanup complete, no orphaned containers found")
        }

        snapshotService := pool.client.SnapshotService("")
        err = snapshotService.Walk(ctx, func(ctx context.Context, info snapshots.Info) error {
                if strings.HasPrefix(info.Name, "ctrd-") {
                        slog.Info("Removing orphaned snapshot", "snapshot_name", info.Name)
                        if err := snapshotService.Remove(ctx, info.Name); err != nil {
                                slog.Error("Failed to remove orphaned snapshot", "snapshot_name", info.Name, "error", err)
                        }
                }
                return nil
        })
        if err != nil {
                slog.Error("Failed to walk snapshots for cleanup", "error", err)
        }

        // close containerd client connection
        if err := pool.client.Close(); err != nil {
                slog.Error("Failed to close containerd client", "error", err)
        }
        pool.client = nil

}
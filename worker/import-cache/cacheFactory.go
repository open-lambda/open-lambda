package cache

/*
TODO:

This is extremely ugly. We should further parameterize the
SBFactories and use them directly instead of repeating code.

*/

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/dockerutil"

	docker "github.com/fsouza/go-dockerclient"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

var unshareFlags []string = []string{"-fimuC"}

const rootCacheSandboxDir = "/tmp/olcache"

func InitCacheFactory(opts *config.Config, cluster string) (cf *BufferedCacheFactory, root sb.ContainerSandbox, rootDir string, err error) {
	cf, root, rootDir, err = NewBufferedCacheFactory(opts, cluster)
	if err != nil {
		return nil, nil, "", err
	}

	return cf, root, rootDir, nil
}

// emptySBInfo wraps sandbox information necessary for the buffer.
type emptySBInfo struct {
	sandbox    sb.ContainerSandbox
	sandboxDir string
}

// BufferedCacheFactory maintains a buffer of sandboxes created by another factory.
type BufferedCacheFactory struct {
	delegate CacheFactory
	buffer   chan *emptySBInfo
	errors   chan error
	dir      string
	idxPtr   *int64
}

type CacheFactory interface {
	Create(sandboxDir string, rootCmd []string) (sb.ContainerSandbox, error)
	Cleanup()
}

// DockerCacheFactory is a SandboxFactory that creates docker sandboxes for the cache.
type DockerCacheFactory struct {
	client  *docker.Client
	cmd     []string
	caps    []string
	labels  map[string]string
	pkgsDir string
}

// NewDockerCacheFactory creates a CacheFactory that uses Docker containers.
func NewDockerCacheFactory(cluster, pkgsDir string) (*DockerCacheFactory, error) {
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

	cf := &DockerCacheFactory{client, cmd, caps, labels, pkgsDir}
	return cf, nil
}

// Create creates a docker container from the pool directory.
func (cf *DockerCacheFactory) Create(sandboxDir string, cmd []string) (sb.ContainerSandbox, error) {
	volumes := []string{
		fmt.Sprintf("%s:%s", sandboxDir, "/host"),
		fmt.Sprintf("%s:%s:ro", cf.pkgsDir, "/packages"),
	}

	container, err := cf.client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:  dockerutil.CACHE_IMAGE,
				Labels: cf.labels,
				Cmd:    cmd,
			},
			HostConfig: &docker.HostConfig{
				Binds:      volumes,
				PidMode:    "host",
				CapAdd:     cf.caps,
				AutoRemove: true,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	sandbox := sb.NewDockerSandbox("", sandboxDir, "", "", container, cf.client)

	return sandbox, nil
}

func (cf *DockerCacheFactory) Cleanup() {
	return
}

// OLContainerCacheFactory is a SandboxFactory that creates olcontainers for the cache.
type OLContainerCacheFactory struct {
	opts    *config.Config
	cgf     *sb.CgroupFactory
	cmd     []string
	baseDir string
	pkgsDir string
}

// NewOLContainerCacheFactory creates a CacheFactory that uses olcontainers.
func NewOLContainerCacheFactory(opts *config.Config, cluster, baseDir, pkgsDir string) (*OLContainerCacheFactory, error) {
	for _, cgroup := range sb.CGroupList {
		cgroupPath := path.Join("/sys/fs/cgroup", cgroup, sb.OLCGroupName)
		if err := os.MkdirAll(cgroupPath, 0700); err != nil {
			return nil, err
		}
	}

	cgf, err := sb.NewCgroupFactory("cache", opts.Cg_pool_size)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(rootCacheSandboxDir, 0777); err != nil {
		return nil, fmt.Errorf("failed to make root sandbox dir :: %v", err.Error())
	} else if err := syscall.Mount(rootCacheSandboxDir, rootCacheSandboxDir, "", sb.BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root sandbox dir: %v", err.Error())
	} else if err := syscall.Mount("none", rootCacheSandboxDir, "", sb.PRIVATE, ""); err != nil {
		return nil, fmt.Errorf("failed to make root cache sandbox dir private :: %v", err.Error())
	}

	return &OLContainerCacheFactory{opts, cgf, []string{"/init"}, baseDir, pkgsDir}, nil
}

// Create creates a docker sandbox from the pool directory.
func (cf *OLContainerCacheFactory) Create(hostDir string, startCmd []string) (sb.ContainerSandbox, error) {
	id_bytes, err := exec.Command("uuidgen").Output()
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(id_bytes[:]))

	rootDir := path.Join(rootCacheSandboxDir, fmt.Sprintf("%s", id))
	if err := os.Mkdir(rootDir, 0700); err != nil {
		return nil, err
	}

	// NOTE: mount points are expected to exist in OLContainer_handler_base directory
	layers := fmt.Sprintf("br=%s=rw:%s=ro", rootDir, cf.baseDir)
	if err := syscall.Mount("none", rootDir, "aufs", 0, layers); err != nil {
		return nil, fmt.Errorf("failed to mount base dir: %v", err.Error())
	}

	sbHostDir := path.Join(rootDir, "host")
	if err := syscall.Mount(hostDir, sbHostDir, "", sb.BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind host dir: %v", err.Error())
	}

	sbPkgsDir := path.Join(rootDir, "packages")
	if err := syscall.Mount(cf.pkgsDir, sbPkgsDir, "", sb.BIND_RO, ""); err != nil {
		return nil, fmt.Errorf("failed to bind packages dir: %v", err.Error())
	}

	unmounts := []string{sbHostDir, sbHostDir, rootDir}
	removals := []string{rootDir}

  umntq := sb.UnmountQueueSingleton()
	sandbox, err := sb.NewOLContainerSandbox(cf.cgf, cf.opts, rootDir, hostDir, id, startCmd, unshareFlags, unmounts, removals, umntq)
	if err != nil {
		return nil, err
	}

	return sandbox, nil
}

func (cf *OLContainerCacheFactory) Cleanup() {
	for _, cgroup := range sb.CGroupList {
		cgroupPath := path.Join("/sys/fs/cgroup", cgroup, sb.OLCGroupName)
		os.Remove(cgroupPath)
	}

	runCmd([]string{"/bin/umount", "/tmp/cache_*/*"})
	runCmd([]string{"/bin/umount", "/tmp/cache_*"})
	runCmd([]string{"/bin/rm", "-rf", "/tmp/cache_*"})
}

// NewBufferedCacheFactory creates a BufferedCacheFactory and starts a go routine to
// fill the sandbox buffer.
func NewBufferedCacheFactory(opts *config.Config, cluster string) (*BufferedCacheFactory, sb.ContainerSandbox, string, error) {
	cacheDir := opts.Import_cache_dir
	pkgsDir := opts.Pkgs_dir
	buffer := opts.Import_cache_buffer
	indexHost := opts.Index_host
	indexPort := opts.Index_port

	rootCmd := []string{"/usr/bin/python", "/server.py"}
	if indexHost != "" && indexPort != "" {
		rootCmd = append(rootCmd, indexHost, indexPort)
	}

	var delegate CacheFactory
	var err error
	if opts.Sandbox == "docker" {
		delegate, err = NewDockerCacheFactory(cluster, pkgsDir)
		if err != nil {
			return nil, nil, "", err
		}
	} else if opts.Sandbox == "olcontainer" {
		delegate, err = NewOLContainerCacheFactory(opts, cluster, opts.OLContainer_cache_base, pkgsDir)
		if err != nil {
			return nil, nil, "", err
		}
	}

	bf := &BufferedCacheFactory{
		delegate: delegate,
		buffer:   make(chan *emptySBInfo, buffer),
		errors:   make(chan error, buffer),
		dir:      cacheDir,
	}

	if err := os.MkdirAll(cacheDir, os.ModeDir); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create pool directory at %s: %v", cacheDir, err)
	}

	// create the root container
	rootDir := filepath.Join(bf.dir, "root")
	if err := os.MkdirAll(rootDir, os.ModeDir); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create cache entry directory at %s: %v", cacheDir, err)
	}

	root, err := bf.delegate.Create(rootDir, rootCmd)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create cache entry sandbox: %v", err)
	} else if err := root.Start(); err != nil {
		return nil, nil, "", fmt.Errorf("failed to start cache entry sandbox: %v", err)
	}

	// fill the sandbox buffer
	var sharedIdx int64 = -1
	bf.idxPtr = &sharedIdx
	for i := 0; i < 5; i++ {
		go func(idxPtr *int64) {
			for {
				newIdx := atomic.AddInt64(idxPtr, 1)
				if newIdx < 0 {
					return // kill signal
				}

				sandboxDir := filepath.Join(bf.dir, fmt.Sprintf("%d", newIdx))
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
			}
		}(bf.idxPtr)
	}

	log.Printf("filling cache buffer")
	for len(bf.buffer) < cap(bf.buffer) {
		time.Sleep(20 * time.Millisecond)
	}
	log.Printf("cache buffer full")

	return bf, root, rootDir, nil
}

// Returns a sandbox ready for a cache interpreter
func (bf *BufferedCacheFactory) Create() (sb.ContainerSandbox, string, error) {
	info, err := <-bf.buffer, <-bf.errors
	if err != nil {
		return nil, "", err
	}

	if err := info.sandbox.Unpause(); err != nil {
		return nil, "", err
	}

	return info.sandbox, info.sandboxDir, nil
}

func (bf *BufferedCacheFactory) Cleanup() {
	// kill signal must be negative for all producers
	atomic.StoreInt64(bf.idxPtr, -1000)

	// empty the buffer
	for {
		select {
		case info := <-bf.buffer:
			if info == nil {
				continue
			}
			info.sandbox.Unpause()
			info.sandbox.Stop()
			info.sandbox.Remove()
		default:
			// clean up mount points once buffer is empty
			bf.delegate.Cleanup()
			return
		}
	}
}

func runCmd(args []string) error {
	c := exec.Cmd{Path: args[0], Args: args}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
}

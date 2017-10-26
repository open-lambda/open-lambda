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

var unshareFlags []string = []string{"-iu"}

const rootCacheSandboxDir = "/tmp/olcache"

type CacheFactory interface {
	Create(parentDir string, startCmd []string) (sb.ContainerSandbox, error)
	Cleanup()
}

func InitCacheFactory(opts *config.Config, cluster string) (cf CacheFactory, root sb.ContainerSandbox, rootDir string, err error) {
	cf, root, rootDir, err = NewCacheFactory(opts, cluster)
	if err != nil {
		return nil, nil, "", err
	}

	return cf, root, rootDir, nil
}

// BufferedCacheFactory maintains a buffer of sandboxes created by another factory.
type BufferedCacheFactory struct {
	delegate CacheFactory
	buffer   chan sb.ContainerSandbox
	errors   chan error
	idxPtr   *int64
}

// DockerCacheFactory is a SandboxFactory that creates docker sandboxes for the cache.
type DockerCacheFactory struct {
	client   *docker.Client
	caps     []string
	labels   map[string]string
	pkgsDir  string
	cacheDir string
	idxPtr   *int64
}

// NewDockerCacheFactory creates a CacheFactory that uses Docker containers.
func NewDockerCacheFactory(cluster, pkgsDir, cacheDir string, idxPtr *int64) (*DockerCacheFactory, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	caps := []string{"SYS_ADMIN"}

	labels := map[string]string{
		dockerutil.DOCKER_LABEL_CLUSTER: cluster,
		dockerutil.DOCKER_LABEL_TYPE:    dockerutil.POOL,
	}

	cf := &DockerCacheFactory{client, caps, labels, pkgsDir, cacheDir, idxPtr}
	return cf, nil
}

// Create creates a docker container from the pool directory.
func (cf *DockerCacheFactory) Create(parentDir string, startCmd []string) (sb.ContainerSandbox, error) {
	newIdx := atomic.AddInt64(cf.idxPtr, 1)
	sandboxDir := filepath.Join(cf.cacheDir, fmt.Sprintf("%d", newIdx))
	if err := os.MkdirAll(sandboxDir, os.ModeDir); err != nil {
		return nil, err
	}

	volumes := []string{
		fmt.Sprintf("%s:%s", sandboxDir, "/host"),
		fmt.Sprintf("%s:%s:ro", cf.pkgsDir, "/packages"),
	}

	container, err := cf.client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:  dockerutil.CACHE_IMAGE,
				Labels: cf.labels,
				Cmd:    startCmd,
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
	opts     *config.Config
	cgf      *sb.CgroupFactory
	baseDir  string
	pkgsDir  string
	cacheDir string
	idxPtr   *int64
}

// NewOLContainerCacheFactory creates a CacheFactory that uses olcontainers.
func NewOLContainerCacheFactory(opts *config.Config, cluster, baseDir, pkgsDir, cacheDir string, idxPtr *int64) (*OLContainerCacheFactory, error) {
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

	sbPkgsDir := path.Join(baseDir, "packages")

	_, err = exec.Command("/bin/sh", "-c", fmt.Sprintf("cp -rT %s %s", pkgsDir, sbPkgsDir)).Output()
	if err != nil {
		log.Printf("failed to copy packages to cache entry base image :: %v", err)
	}

	return &OLContainerCacheFactory{opts, cgf, baseDir, pkgsDir, cacheDir, idxPtr}, nil
}

// Create creates a docker sandbox from the pool directory.
func (cf *OLContainerCacheFactory) Create(parentDir string, startCmd []string) (sb.ContainerSandbox, error) {
	newIdx := atomic.AddInt64(cf.idxPtr, 1)
	hostDir := filepath.Join(cf.cacheDir, fmt.Sprintf("%d", newIdx))
	if err := os.MkdirAll(hostDir, os.ModeDir); err != nil {
		return nil, err
	}
	// pipe for synchronization before init is ready
	pipe := filepath.Join(hostDir, "init_pipe")
	if err := syscall.Mkfifo(pipe, 0777); err != nil {
		return nil, err
	}
	// pipe for synchronization before socket is ready
	pipe = filepath.Join(hostDir, "server_pipe")
	if err := syscall.Mkfifo(pipe, 0777); err != nil {
		return nil, err
	}

	id_bytes, err := exec.Command("uuidgen").Output()
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(id_bytes[:]))

	/*
		var rootDir string
		if parentDir == "" {
	*/
	rootDir := filepath.Join(rootCacheSandboxDir, fmt.Sprintf("cache_%s", id))
	/*
		} else {
			rootDir = filepath.Join(parentDir, "tmp", fmt.Sprintf("cache_%s", id))
		}
	*/

	if err := os.Mkdir(rootDir, 0700); err != nil {
		return nil, err
	}

	// NOTE: mount points are expected to exist in OLContainer_handler_base directory

	if err := syscall.Mount(cf.baseDir, rootDir, "", sb.BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root dir: %s -> %s :: %v\n", cf.baseDir, rootDir, err)
	} else if err := syscall.Mount("none", rootDir, "", sb.BIND_RO, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root dir RO: %s :: %v\n", rootDir, err)
	} else if err := syscall.Mount("none", rootDir, "", sb.PRIVATE, ""); err != nil {
		return nil, fmt.Errorf("failed to make root dir private :: %v", err)
	}

	sandbox, err := sb.NewOLContainerSandbox(cf.cgf, cf.opts, rootDir, id, startCmd, unshareFlags)
	if err != nil {
		return nil, err
	}

	if err := sandbox.MountDirs(hostDir, ""); err != nil {
		sandbox.Stop()
		sandbox.Remove()
		return nil, err
	}

	if err := sandbox.Start(); err != nil {
		sandbox.Stop()
		sandbox.Remove()
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

// NewCacheFactory creates a BufferedCacheFactory and starts a go routine to
// fill the sandbox buffer.
func NewCacheFactory(opts *config.Config, cluster string) (CacheFactory, sb.ContainerSandbox, string, error) {
	cacheDir := opts.Import_cache_dir
	pkgsDir := opts.Pkgs_dir
	buffer := opts.Import_cache_buffer
	indexHost := opts.Index_host
	indexPort := opts.Index_port

	if err := os.MkdirAll(cacheDir, os.ModeDir); err != nil {
		return nil, nil, "", fmt.Errorf("failed to create pool directory at %s: %v", cacheDir, err)
	}

	rootCmd := []string{"/usr/bin/python", "/server.py"}
	if indexHost != "" && indexPort != "" {
		rootCmd = append(rootCmd, indexHost, indexPort)
	}

	var sharedIdx int64 = -1
	idxPtr := &sharedIdx

	var delegate CacheFactory
	var err error
	if opts.Sandbox == "docker" {
		delegate, err = NewDockerCacheFactory(cluster, pkgsDir, cacheDir, idxPtr)
		if err != nil {
			return nil, nil, "", err
		}
	} else if opts.Sandbox == "olcontainer" {
		delegate, err = NewOLContainerCacheFactory(opts, cluster, opts.OLContainer_cache_base, pkgsDir, cacheDir, idxPtr)
		if err != nil {
			return nil, nil, "", err
		}
	}

	// create the root container
	root, err := delegate.Create("", rootCmd)
	if err != nil {
		return nil, nil, "", fmt.Errorf("failed to create cache entry sandbox: %v", err)
	}

	rootDir := filepath.Join(cacheDir, "0")
	if buffer == 0 {
		return delegate, root, rootDir, nil
	}

	bf := &BufferedCacheFactory{
		delegate: delegate,
		buffer:   make(chan sb.ContainerSandbox, buffer),
		errors:   make(chan error, buffer),
		idxPtr:   idxPtr,
	}

	threads := 1
	if opts.Import_cache_buffer_threads > 0 {
		threads = opts.Import_cache_buffer_threads
	}

	for i := 0; i < threads; i++ {
		go func(idxPtr *int64) {
			for {
				if atomic.LoadInt64(idxPtr) < 0 {
					return // kill signal
				}

				// expect sandbox to come back started
				if sandbox, err := bf.delegate.Create("", []string{"/init"}); err != nil {
					bf.buffer <- nil
					bf.errors <- err
				} else {
					bf.buffer <- sandbox
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
func (bf *BufferedCacheFactory) Create(rootDir string, startCmd []string) (sb.ContainerSandbox, error) {
	sandbox, err := <-bf.buffer, <-bf.errors
	if err != nil {
		return nil, err
	}

	return sandbox, nil
}

func (bf *BufferedCacheFactory) Cleanup() {
	// kill signal must be negative for all producers
	atomic.StoreInt64(bf.idxPtr, -1000000)

	// empty the buffer
	for {
		select {
		case sandbox := <-bf.buffer:
			if sandbox == nil {
				continue
			}
			sandbox.Unpause()
			sandbox.Stop()
			sandbox.Remove()
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

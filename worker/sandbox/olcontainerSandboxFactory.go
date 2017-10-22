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

	"github.com/open-lambda/open-lambda/worker/config"
)

const rootSandboxDir string = "/tmp/olsbs"

var BIND uintptr = uintptr(syscall.MS_BIND)
var BIND_RO uintptr = uintptr(syscall.MS_BIND | syscall.MS_RDONLY | syscall.MS_REMOUNT)
var PRIVATE uintptr = uintptr(syscall.MS_PRIVATE)
var SHARED uintptr = uintptr(syscall.MS_SHARED)

var unshareFlags []string = []string{"-impuf", "--propagation", "slave"}

// OLContainerSBFactory is a SandboxFactory that creats docker sandboxes.
type OLContainerSBFactory struct {
	opts      *config.Config
	cgf       *CgroupFactory
	baseDir   string
	pkgsDir   string
	indexHost string
	indexPort string
}

// NewOLContainerSBFactory creates a OLContainerSBFactory.
func NewOLContainerSBFactory(opts *config.Config) (*OLContainerSBFactory, error) {
	for _, cgroup := range CGroupList {
		cgroupPath := filepath.Join("/sys/fs/cgroup", cgroup, OLCGroupName)
		if err := os.MkdirAll(cgroupPath, 0700); err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(rootSandboxDir, 0777); err != nil {
		return nil, fmt.Errorf("failed to make root sandbox dir :: %v", err)
	} else if err := syscall.Mount(rootSandboxDir, rootSandboxDir, "", BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root sandbox dir: %v", err)
	} else if err := syscall.Mount("none", rootSandboxDir, "", PRIVATE, ""); err != nil {
		return nil, fmt.Errorf("failed to make root sandbox dir private :: %v", err)
	}

	baseDir := opts.OLContainer_handler_base

	pkgsDir := filepath.Join(baseDir, "packages")
	if err := syscall.Mount(opts.Pkgs_dir, pkgsDir, "", BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind packages dir: %s -> %s :: %v", opts.Pkgs_dir, pkgsDir, err)
	} else if err := syscall.Mount("none", pkgsDir, "", BIND_RO, ""); err != nil {
		return nil, fmt.Errorf("failed to bind packages dir RO: %s -> %s :: %v", opts.Pkgs_dir, pkgsDir, err)
	}

	cgf, err := NewCgroupFactory("sandbox", opts.Cg_pool_size)
	if err != nil {
		return nil, err
	}

	sf := &OLContainerSBFactory{
		opts:      opts,
		cgf:       cgf,
		baseDir:   baseDir,
		pkgsDir:   pkgsDir,
		indexHost: opts.Index_host,
		indexPort: opts.Index_port,
	}

	return sf, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (sf *OLContainerSBFactory) Create(handlerDir, workingDir string) (Sandbox, error) {
	id_bytes, err := exec.Command("uuidgen").Output()
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(id_bytes[:]))

	rootDir := filepath.Join(rootSandboxDir, id)
	if err := os.Mkdir(rootDir, 0777); err != nil {
		return nil, err
	}

	startCmd := []string{"/ol-init"}
	if sf.indexHost != "" {
		startCmd = append(startCmd, sf.indexHost)
	}
	if sf.indexPort != "" {
		startCmd = append(startCmd, sf.indexPort)
	}

	// NOTE: mount points are expected to exist in OLContainer_handler_base directory

	if err := syscall.Mount(sf.baseDir, rootDir, "", BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root dir: %s -> %s :: %v\n", sf.baseDir, rootDir, err)
	} else if err := syscall.Mount("none", rootDir, "", BIND_RO, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root dir RO: %s :: %v\n", rootDir, err)
	} else if err := syscall.Mount("none", rootDir, "", PRIVATE, ""); err != nil {
		return nil, fmt.Errorf("failed to make root dir private :: %v", err)
	}

	sandbox, err := NewOLContainerSandbox(sf.cgf, sf.opts, rootDir, id, startCmd, unshareFlags)
	if err != nil {
		return nil, err
	}

	// if using buffer we will mount the rest of the directories later
	if handlerDir == "" && workingDir == "" {
		sbHostDir := filepath.Join(rootDir, "host")
		if err := syscall.Mount(sbHostDir, sbHostDir, "", BIND, ""); err != nil {
			return nil, fmt.Errorf("failed to bind sandbox host dir onto itself :: %v\n", err)
		} else if err := syscall.Mount("none", sbHostDir, "", SHARED, ""); err != nil {
			return nil, fmt.Errorf("failed to make sbHostDir shared :: %v\n", err)
		}

		sbHandlerDir := filepath.Join(rootDir, "handler")
		if err := syscall.Mount(sbHandlerDir, sbHandlerDir, "", BIND, ""); err != nil {
			return nil, fmt.Errorf("failed to bind sbHandlerDir onto itself :: %v\n", err)
		} else if err := syscall.Mount("none", sbHandlerDir, "", SHARED, ""); err != nil {
			return nil, fmt.Errorf("failed to make sbHandlerDir shared :: %v\n", err)
		}

		sbTmpDir := filepath.Join(rootDir, "tmp")
		if err := syscall.Mount(sbTmpDir, sbTmpDir, "", BIND, ""); err != nil {
			return nil, fmt.Errorf("failed to bind sbTmpDir onto itself :: %v\n", err)
		} else if err := syscall.Mount("none", sbTmpDir, "", SHARED, ""); err != nil {
			return nil, fmt.Errorf("failed to make sbTmpDir shared :: %v\n", err)
		}

		return sandbox, nil
	}

	// create sandbox directories
	hostDir := filepath.Join(workingDir, id)
	if err := os.MkdirAll(hostDir, 0777); err != nil {
		return nil, err
	}

	if err := sandbox.MountDirs(hostDir, handlerDir); err != nil {
		return nil, err
	}

	return sandbox, nil
}

func (sf *OLContainerSBFactory) Cleanup() {
	for _, cgroup := range CGroupList {
		cgroupPath := filepath.Join("/sys/fs/cgroup", cgroup, OLCGroupName)
		os.RemoveAll(cgroupPath)
	}

	syscall.Unmount(sf.pkgsDir, syscall.MNT_DETACH)
	syscall.Unmount(rootSandboxDir, syscall.MNT_DETACH)
	os.RemoveAll(rootSandboxDir)
}

// BufferedOLContainerSBFactory maintains a buffer of sandboxes created by another factory.
type BufferedOLContainerSBFactory struct {
	delegate SandboxFactory
	buffer   chan *OLContainerSandbox
	errors   chan error
	cache    bool
	idxPtr   *int64
}

// NewBufferedOLContainerSBFactory creates a BufferedOLContainerSBFactory and starts a go routine to
// fill the sandbox buffer.
func NewBufferedOLContainerSBFactory(opts *config.Config, delegate SandboxFactory) (*BufferedOLContainerSBFactory, error) {
	bf := &BufferedOLContainerSBFactory{
		delegate: delegate,
		buffer:   make(chan *OLContainerSandbox, opts.Sandbox_buffer),
		errors:   make(chan error, opts.Sandbox_buffer),
		cache:    opts.Import_cache_size != 0,
	}

	if err := os.MkdirAll(olMntDir, os.ModeDir); err != nil {
		return nil, fmt.Errorf("fail to create directory at %s: %v", olMntDir, err)
	}

	// fill the sandbox buffer
	start := time.Now()
	log.Printf("filling sandbox buffer")

	var sharedIdx int64 = -1
	bf.idxPtr = &sharedIdx
	for i := 0; i < 20; i++ {
		go func(idxPtr *int64) {
			for {
				newIdx := atomic.AddInt64(idxPtr, 1)
				if newIdx < 0 {
					return // kill signal
				}

				if sandbox, err := bf.delegate.Create("", ""); err != nil {
					bf.errors <- err
				} else if sandbox, ok := sandbox.(*OLContainerSandbox); !ok {
					bf.errors <- err
				} else if err := sandbox.Start(); err != nil {
					bf.errors <- err
				} else if err := sandbox.Pause(); err != nil {
					bf.errors <- err
				} else {
					bf.buffer <- sandbox
				}
			}
		}(bf.idxPtr)
	}

	for len(bf.buffer) < cap(bf.buffer) {
		time.Sleep(20 * time.Millisecond)
	}
	log.Printf("sandbox buffer full in %v", time.Since(start))

	return bf, nil
}

// Create mounts the handler and sandbox directories to the ones already
// mounted in the sandbox, and returns that sandbox. The sandbox would be in
// Paused state, instead of Stopped.
func (bf *BufferedOLContainerSBFactory) Create(handlerDir, workingDir string) (Sandbox, error) {
	select {
	case sandbox := <-bf.buffer:
		// create cluster host directory
		hostDir := filepath.Join(workingDir, sandbox.id)
		if err := os.MkdirAll(hostDir, 0777); err != nil {
			return nil, err
		}

		if err := sandbox.MountDirs(hostDir, handlerDir); err != nil {
			return nil, err
		}

		return sandbox, nil

	case err := <-bf.errors:
		return nil, err
	}
}

func (bf *BufferedOLContainerSBFactory) Cleanup() {
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
		case sandbox := <-bf.buffer:
			if sandbox == nil {
				continue
			}
			sandbox.Unpause()
			sandbox.Stop()
			sandbox.Remove()

		default:
			bf.delegate.Cleanup()

			return
		}
	}
}

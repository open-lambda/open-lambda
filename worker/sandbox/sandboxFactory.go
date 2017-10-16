package sandbox

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
)

const olMntDir = "/tmp/olmnts"

// SandboxFactory is the common interface for all sandbox creation functions.
type SandboxFactory interface {
	Create(handlerDir, workingDir, indexHost, indexPort string) (sandbox Sandbox, err error)
	Cleanup()
}

// BufferedSBFactory maintains a buffer of sandboxes created by another factory.
type BufferedSBFactory struct {
	delegate SandboxFactory
	buffer   chan *emptySBInfo
	errors   chan error
	mntDir   string
	cache    bool
	idxPtr   *int64
}

// emptySBInfo wraps sandbox information necessary for the buffer.
type emptySBInfo struct {
	sandbox    Sandbox
	handlerDir string
	sandboxDir string
}

func InitSandboxFactory(config *config.Config) (sf SandboxFactory, err error) {
	var delegate SandboxFactory
	if config.Sandbox == "docker" {
		delegate, err = NewDockerSBFactory(config)
		if err != nil {
			return nil, err
		}
	} else if config.Sandbox == "olcontainer" {
		delegate, err = NewOLContainerSBFactory(config, config.OLContainer_handler_base)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("invalid sandbox type: '%s'", config.Sandbox)
	}

	if config.Sandbox_buffer == 0 {
		return delegate, nil
	}

	return NewBufferedSBFactory(config, delegate)
}

// mkSBDirs makes the handler and sandbox directories and tries to unmount them.
func mkSBDirs(bufDir string) (string, string, error) {
	if err := os.MkdirAll(bufDir, os.ModeDir); err != nil {
		return "", "", fmt.Errorf("fail to create directory at %s: %v", bufDir, err)
	}
	handlerDir := filepath.Join(bufDir, "handler")
	if err := os.MkdirAll(handlerDir, os.ModeDir); err != nil {
		return "", "", fmt.Errorf("fail to create directory at %s: %v", handlerDir, err)
	}
	if err := syscall.Unmount(handlerDir, 0); err != nil && err != syscall.EINVAL {
		return "", "", fmt.Errorf("fail to unmount directory %s: %v", handlerDir, err)
	}
	sandboxDir := filepath.Join(bufDir, "host")
	if err := os.MkdirAll(sandboxDir, os.ModeDir); err != nil {
		return "", "", fmt.Errorf("fail to create directory at %s: %v", sandboxDir, err)
	}
	if err := syscall.Unmount(sandboxDir, 0); err != nil && err != syscall.EINVAL {
		return "", "", fmt.Errorf("fail to unmount directory %s: %v", sandboxDir, err)
	}
	return handlerDir, sandboxDir, nil
}

// NewBufferedSBFactory creates a BufferedSBFactory and starts a go routine to
// fill the sandbox buffer.
func NewBufferedSBFactory(opts *config.Config, delegate SandboxFactory) (*BufferedSBFactory, error) {
	bf := &BufferedSBFactory{}
	bf.delegate = delegate
	bf.buffer = make(chan *emptySBInfo, opts.Sandbox_buffer)
	bf.errors = make(chan error, opts.Sandbox_buffer)
	if opts.Import_cache_size == 0 {
		bf.cache = false
	} else {
		bf.cache = true
	}

	if err := os.MkdirAll(olMntDir, os.ModeDir); err != nil {
		return nil, fmt.Errorf("fail to create directory at %s: %v", olMntDir, err)
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

				bufDir := filepath.Join(olMntDir, fmt.Sprintf("%d", newIdx))
				if handlerDir, sandboxDir, err := mkSBDirs(bufDir); err != nil {
					bf.errors <- err
				} else if sandbox, err := bf.delegate.Create(handlerDir, sandboxDir, opts.Index_host, opts.Index_port); err != nil {
					bf.errors <- err
				} else if err := sandbox.Start(); err != nil {
					bf.errors <- err
				} else if err := sandbox.Pause(); err != nil {
					bf.errors <- err
				} else {
					bf.buffer <- &emptySBInfo{sandbox, handlerDir, sandboxDir}
				}
			}
		}(bf.idxPtr)
	}

	log.Printf("filling sandbox buffer")
	for len(bf.buffer) < cap(bf.buffer) {
		time.Sleep(20 * time.Millisecond)
	}
	log.Printf("sandbox buffer full")

	return bf, nil
}

// Create mounts the handler and sandbox directories to the ones already
// mounted in the sandbox, and returns that sandbox. The sandbox would be in
// Paused state, instead of Stopped.
func (bf *BufferedSBFactory) Create(handlerDir, workingDir, indexHost, indexPort string) (Sandbox, error) {
	mntFlag := uintptr(syscall.MS_BIND)
	select {
	case info := <-bf.buffer:
		// create cluster host directory
		hostDir := path.Join(workingDir, info.sandbox.ID())
		if err := os.MkdirAll(hostDir, 0777); err != nil {
			return nil, err
		}

		if err := info.sandbox.Unpause(); err != nil {
			return nil, err
		} else if err := syscall.Mount(handlerDir, info.handlerDir, "", mntFlag, ""); err != nil {
			return nil, err
		} else if err := syscall.Mount(hostDir, path.Join(info.sandboxDir, info.sandbox.ID()), "", mntFlag, ""); err != nil {
			return nil, err
		}
		if !bf.cache {
			sockPath := filepath.Join(workingDir, "ol.sock")
			_ = os.Remove(sockPath)
		}

		return info.sandbox, nil

	case err := <-bf.errors:
		return nil, err
	}
}

func (bf *BufferedSBFactory) Cleanup() {
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
			bf.delegate.Cleanup()

			// clean up directories once all sandboxes are dead
			runCmd([]string{"umount", filepath.Join(olMntDir, "*", "*")})
			runCmd([]string{"rm", "-rf", olMntDir})

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

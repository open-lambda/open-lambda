package sandbox

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
)

const rootSandboxDir string = "/tmp/olsbs"

var BIND uintptr = uintptr(syscall.MS_BIND)
var BIND_RO uintptr = uintptr(syscall.MS_BIND | syscall.MS_RDONLY | syscall.MS_REMOUNT)
var PRIVATE uintptr = uintptr(syscall.MS_PRIVATE)
var SHARED uintptr = uintptr(syscall.MS_SHARED)

var unshareFlags []string = []string{"-ipu"}

// SOCKSBFactory is a SandboxFactory that creats docker sandboxes.
type SOCKSBFactory struct {
	opts      *config.Config
	cgf       *CgroupFactory
	baseDir   string
	pkgsDir   string
	indexHost string
	indexPort string
}

// NewSOCKSBFactory creates a SOCKSBFactory.
func NewSOCKSBFactory(opts *config.Config) (*SOCKSBFactory, error) {
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

	baseDir := opts.SOCK_handler_base
	pkgsDir := filepath.Join(baseDir, "packages")

	_, err := exec.Command("/bin/sh", "-c", fmt.Sprintf("cp -rT %s %s", opts.Pkgs_dir, pkgsDir)).Output()
	if err != nil {
		log.Printf("failed to copy packages to lambda base image :: %v", err)
	}

	cgf, err := NewCgroupFactory("sandbox", opts.Cg_pool_size)
	if err != nil {
		return nil, err
	}

	sf := &SOCKSBFactory{
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
func (sf *SOCKSBFactory) Create(handlerDir, workingDir string) (Sandbox, error) {
	if config.Timing {
		defer func(start time.Time) {
			log.Printf("create sock took %v\n", time.Since(start))
		}(time.Now())
	}

	id_bytes, err := exec.Command("uuidgen").Output()
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(id_bytes[:]))

	rootDir := filepath.Join(rootSandboxDir, fmt.Sprintf("sb_%s", id))
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

	// NOTE: mount points are expected to exist in SOCK_handler_base directory

	if err := syscall.Mount(sf.baseDir, rootDir, "", BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root dir: %s -> %s :: %v\n", sf.baseDir, rootDir, err)
	} else if err := syscall.Mount("none", rootDir, "", BIND_RO, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root dir RO: %s :: %v\n", rootDir, err)
	} else if err := syscall.Mount("none", rootDir, "", PRIVATE, ""); err != nil {
		return nil, fmt.Errorf("failed to make root dir private :: %v", err)
	}

	sandbox, err := NewSOCKSandbox(sf.cgf, sf.opts, rootDir, id, startCmd, unshareFlags)
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

		return sandbox, nil
	}

	// create sandbox directories
	hostDir := filepath.Join(workingDir, id)
	if err := os.MkdirAll(hostDir, 0777); err != nil {
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

	if err := sandbox.MountDirs(hostDir, handlerDir); err != nil {
		return nil, err
	}

	return sandbox, nil
}

func (sf *SOCKSBFactory) Cleanup() {
	for _, cgroup := range CGroupList {
		cgroupPath := filepath.Join("/sys/fs/cgroup", cgroup, OLCGroupName)
		os.RemoveAll(cgroupPath)
	}

	syscall.Unmount(sf.pkgsDir, syscall.MNT_DETACH)
	syscall.Unmount(rootSandboxDir, syscall.MNT_DETACH)
	os.RemoveAll(rootSandboxDir)
}

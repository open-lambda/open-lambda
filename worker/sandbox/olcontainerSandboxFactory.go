package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"syscall"

	"github.com/open-lambda/open-lambda/worker/config"
)

const rootSandboxDir string = "/tmp/olsbs"

var BIND uintptr = uintptr(syscall.MS_BIND)
var BIND_RO uintptr = uintptr(syscall.MS_BIND | syscall.MS_RDONLY | syscall.MS_REMOUNT)
var PRIVATE uintptr = uintptr(syscall.MS_PRIVATE)

var unshareFlags []string = []string{"-impuf", "--propagation", "slave"}

// OLContainerSBFactory is a SandboxFactory that creats docker sandboxes.
type OLContainerSBFactory struct {
	mutex   sync.Mutex
	opts    *config.Config
	baseDir string
	cgf     *CgroupFactory
}

// NewOLContainerSBFactory creates a OLContainerSBFactory.
func NewOLContainerSBFactory(opts *config.Config, baseDir string) (*OLContainerSBFactory, error) {
	for _, cgroup := range CGroupList {
		cgroupPath := path.Join("/sys/fs/cgroup", cgroup, OLCGroupName)
		if err := os.MkdirAll(cgroupPath, 0700); err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(rootSandboxDir, 0777); err != nil {
		return nil, fmt.Errorf("failed to make root sandbox dir :: %v", err.Error())
	} else if err := syscall.Mount(rootSandboxDir, rootSandboxDir, "", BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root sandbox dir: %v", err.Error())
	} else if err := syscall.Mount("none", rootSandboxDir, "", PRIVATE, ""); err != nil {
		return nil, fmt.Errorf("failed to make root sandbox dir private :: %v", err.Error())
	}

	cgf, err := NewCgroupFactory("sandbox", opts.Cg_pool_size)
	if err != nil {
		return nil, err
	}

	return &OLContainerSBFactory{opts: opts, baseDir: baseDir, cgf: cgf}, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (sf *OLContainerSBFactory) Create(handlerDir, workingDir, indexHost, indexPort string) (Sandbox, error) {
	id_bytes, err := exec.Command("uuidgen").Output()
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(id_bytes[:]))

	// create sandbox directories
	hostDir := path.Join(workingDir, id)
	if err := os.MkdirAll(hostDir, 0777); err != nil {
		return nil, err
	}

	pipDir := path.Join(hostDir, "pip")
	if err := os.Mkdir(pipDir, 0777); err != nil {
		return nil, err
	}

	tmpDir := path.Join(hostDir, "tmp")
	if err := os.Mkdir(tmpDir, 0777); err != nil {
		return nil, err
	}

	rootDir := path.Join(rootSandboxDir, id)
	if err := os.Mkdir(rootDir, 0700); err != nil {
		return nil, err
	}

	// NOTE: mount points are expected to exist in OLContainer_handler_base directory

	/*
		layers := fmt.Sprintf("br=%s=rw:%s=ro", rootDir, sf.baseDir)
		if err := syscall.Mount("none", rootDir, "aufs", 0, layers); err != nil {
			return nil, fmt.Errorf("failed to mount base dir: %v", err.Error())
		} else if err := syscall.Mount("none", rootDir, "", PRIVATE, ""); err != nil {
			return nil, fmt.Errorf("failed to make root dir private :: %v", err.Error())
		}
	*/
	sf.mutex.Lock()
	if err := syscall.Mount(sf.baseDir, rootDir, "", BIND, ""); err != nil {
		sf.mutex.Unlock()
		return nil, fmt.Errorf("failed to bind root dir: %s -> %s :: %v\n", sf.baseDir, rootDir, err.Error())
	} else if err := syscall.Mount(sf.baseDir, rootDir, "", BIND_RO, ""); err != nil {
		sf.mutex.Unlock()
		return nil, fmt.Errorf("failed to bind root dir RO: %s :: %v\n", rootDir, err.Error())
	}

	sbHandlerDir := path.Join(rootDir, "handler")
	if err := syscall.Mount(handlerDir, sbHandlerDir, "", BIND, ""); err != nil {
		sf.mutex.Unlock()
		return nil, fmt.Errorf("failed to bind handler dir: %s -> %s :: %v", handlerDir, sbHandlerDir, err.Error())
	} else if err := syscall.Mount(handlerDir, sbHandlerDir, "", BIND_RO, ""); err != nil {
		sf.mutex.Unlock()
		return nil, fmt.Errorf("failed to bind handler dir RO: %v", err.Error())
	}

	sbHostDir := path.Join(rootDir, "host")
	if err := syscall.Mount(hostDir, sbHostDir, "", BIND, ""); err != nil {
		sf.mutex.Unlock()
		return nil, fmt.Errorf("failed to bind host dir: %v", err.Error())
	}

	sbTmpDir := path.Join(rootDir, "tmp")
	if err := syscall.Mount(tmpDir, sbTmpDir, "", BIND, ""); err != nil {
		sf.mutex.Unlock()
		return nil, fmt.Errorf("failed to bind tmp dir: %v", err.Error())
	}

	sbPkgsDir := path.Join(rootDir, "packages")
	if err := syscall.Mount(sf.opts.Pkgs_dir, sbPkgsDir, "", BIND, ""); err != nil {
		sf.mutex.Unlock()
		return nil, fmt.Errorf("failed to bind packages dir: %s -> %s :: %v", sf.opts.Pkgs_dir, sbPkgsDir, err.Error())
	} else if err := syscall.Mount(sf.opts.Pkgs_dir, sbPkgsDir, "", BIND_RO, ""); err != nil {
		sf.mutex.Unlock()
		return nil, fmt.Errorf("failed to bind packages dir RO: %s -> %s :: %v", sf.opts.Pkgs_dir, sbPkgsDir, err.Error())
	}
	sf.mutex.Unlock()

	startCmd := []string{"/ol-init"}
	if indexHost != "" {
		startCmd = append(startCmd, indexHost)
	}
	if indexPort != "" {
		startCmd = append(startCmd, indexPort)
	}

	unmounts := []string{rootDir}
	removals := []string{rootDir}

	return NewOLContainerSandbox(sf.cgf, sf.opts, rootDir, hostDir, id, startCmd, unshareFlags, unmounts, removals)
}

func (sf *OLContainerSBFactory) Cleanup() {
	for _, cgroup := range CGroupList {
		cgroupPath := path.Join("/sys/fs/cgroup", cgroup, OLCGroupName)
		os.RemoveAll(cgroupPath)
	}

	//log.Printf("cleanup, unmount root dir: %v", syscall.Unmount(rootSandboxDir, syscall.MNT_DETACH))
	//log.Printf("cleanup, remove root dir: %v", os.RemoveAll(rootSandboxDir))
}

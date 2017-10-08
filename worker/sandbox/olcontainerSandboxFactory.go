package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	"github.com/open-lambda/open-lambda/worker/config"
)

var unshareFlags []string = []string{"-impuf", "--mount-proc", "--propagation", "unchanged"}

// OLContainerSBFactory is a SandboxFactory that creats docker sandboxes.
type OLContainerSBFactory struct {
	opts    *config.Config
	baseDir string
}

// NewOLContainerSBFactory creates a OLContainerSBFactory.
func NewOLContainerSBFactory(opts *config.Config, baseDir string) (*OLContainerSBFactory, error) {
	for _, cgroup := range CGroupList {
		cgroupPath := path.Join("/sys/fs/cgroup", cgroup, OLCGroupName)
		if err := os.MkdirAll(cgroupPath, 0700); err != nil {
			return nil, err
		}
	}

	return &OLContainerSBFactory{opts: opts, baseDir: baseDir}, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (sf *OLContainerSBFactory) Create(handlerDir, workingDir, indexHost, indexPort string) (Sandbox, error) {
	id_bytes, err := exec.Command("uuidgen").Output()
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(id_bytes[:]))

	// create sandbox directory
	hostDir := path.Join(workingDir, id)
	if err := os.MkdirAll(hostDir, 0777); err != nil {
		return nil, err
	}

	rootDir := fmt.Sprintf("/tmp/sandbox_%s", id)
	if err := os.Mkdir(rootDir, 0700); err != nil {
		return nil, err
	}

	BIND_RO := uintptr(syscall.MS_BIND | syscall.MS_RDONLY)

	// NOTE: mount points are expected to exist in OLContainer_handler_base directory

	layers := fmt.Sprintf("br=%s=rw:%s=ro", rootDir, sf.baseDir)
	if err := syscall.Mount("none", rootDir, "aufs", 0, layers); err != nil {
		return nil, fmt.Errorf("failed to mount base dir: %v", err.Error())
	}

	containerHandlerDir := path.Join(rootDir, "handler")
	if err := syscall.Mount(handlerDir, containerHandlerDir, "", syscall.MS_BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind handler dir: %v", err.Error())
	} else if err := syscall.Mount("none", containerHandlerDir, "", syscall.MS_SLAVE, ""); err != nil {
		return nil, fmt.Errorf("failed to make handler dir a slave: %v", err.Error())
	}

	sbHostDir := path.Join(rootDir, "host")
	if err := syscall.Mount(hostDir, sbHostDir, "", syscall.MS_BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind host dir: %v", err.Error())
	} else if err := syscall.Mount("none", sbHostDir, "", syscall.MS_SLAVE, ""); err != nil {
		return nil, fmt.Errorf("failed to make host dir a slave: %v", err.Error())
	}

	pkgsDir := path.Join(rootDir, "packages")
	if err := syscall.Mount(sf.opts.Pkgs_dir, pkgsDir, "", BIND_RO, ""); err != nil {
		return nil, fmt.Errorf("failed to bind handler dir: %v", err.Error())
	}

	startCmd := []string{"/ol-init"}
	if indexHost != "" {
		startCmd = append(startCmd, indexHost)
	}
	if indexPort != "" {
		startCmd = append(startCmd, indexPort)
	}

	return NewOLContainerSandbox(sf.opts, rootDir, hostDir, id, startCmd, unshareFlags)
}

func (sf *OLContainerSBFactory) Cleanup() {
	for _, cgroup := range CGroupList {
		cgroupPath := path.Join("/sys/fs/cgroup", cgroup, OLCGroupName)
		os.Remove(cgroupPath)
	}
}

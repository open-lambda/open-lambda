package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/open-lambda/open-lambda/worker/config"
)

// OLContainerSBFactory is a SandboxFactory that creats docker sandboxes.
type OLContainerSBFactory struct {
	opts *config.Config
}

// NewOLContainerSBFactory creates a OLContainerSBFactory.
func NewOLContainerSBFactory(opts *config.Config) (*OLContainerSBFactory, error) {
	for _, cgroup := range cgroupList {
		cgroupPath := path.Join("/sys/fs/cgroup", cgroup, olCGroupName)
		if err := os.MkdirAll(cgroupPath, 0700); err != nil {
			return nil, err
		}
	}

	return &OLContainerSBFactory{opts: opts}, nil
}

func cmd(args []string) error {
	c := exec.Cmd{Path: args[0], Args: args}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	return err
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (sf *OLContainerSBFactory) Create(handlerDir, sandboxDir, indexHost, indexPort string) (Sandbox, error) {
	id_bytes, err := exec.Command("uuidgen").Output()
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(id_bytes[:]))

	rootDir := path.Join(fmt.Sprintf("/tmp/sandbox_%s", id))
	if err := os.Mkdir(rootDir, 0700); err != nil {
		return nil, err
	}

	// NOTE: mount points are expected to exist in OLContainer_base directory
	err = cmd([]string{"/bin/mount", "--bind", "-o", "ro", sf.opts.OLContainer_base, rootDir})
	if err != nil {
		return nil, fmt.Errorf("Failed to bind base: %v", err.Error())
	}

	err = cmd([]string{"/bin/mount", "--bind", "-o", "ro", handlerDir, path.Join(rootDir, "handler")})
	if err != nil {
		return nil, fmt.Errorf("Failed to bind handler dir: %v", err.Error())
	}

	err = cmd([]string{"/bin/mount", "--bind", "-o", "ro", sf.opts.Pkgs_dir, path.Join(rootDir, "packages")})
	if err != nil {
		return nil, fmt.Errorf("Failed to bind packages dir: %v", err.Error())
	}

	err = cmd([]string{"/bin/mount", "--bind", sandboxDir, path.Join(rootDir, "host")})
	if err != nil {
		return nil, fmt.Errorf("Failed to bind host dir: %v", err.Error())
	}

	sandbox, err := NewOLContainerSandbox(sf.opts, rootDir, indexHost, indexPort, id)
	if err != nil {
		return nil, err
	}

	return sandbox, nil
}

// TODO
func (sf *OLContainerSBFactory) Cleanup() {
	for _, cgroup := range cgroupList {
		cgroupPath := path.Join("/sys/fs/cgroup", cgroup, olCGroupName)
		os.Remove(cgroupPath)
	}
}

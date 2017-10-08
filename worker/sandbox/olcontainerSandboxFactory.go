package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/open-lambda/open-lambda/worker/config"
)

var unshareFlags []string = []string{"-impuf", "--mount-proc"}

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

func runCmd(args []string) error {
	c := exec.Cmd{Path: args[0], Args: args}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
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

	// NOTE: mount points are expected to exist in OLContainer_handler_base directory
	layers := fmt.Sprintf("br=%s=rw:%s=ro", rootDir, sf.baseDir)
	err = runCmd([]string{"/bin/mount", "-t", "aufs", "-o", layers, "none", rootDir})
	if err != nil {
		return nil, fmt.Errorf("Failed to bind base: %v", err.Error())
	}

	err = runCmd([]string{"/bin/mount", "--bind", "-o", "ro", handlerDir, path.Join(rootDir, "handler")})
	if err != nil {
		return nil, fmt.Errorf("Failed to bind handler dir: %v", err.Error())
	}

	err = runCmd([]string{"/bin/mount", "--bind", "-o", "ro", sf.opts.Pkgs_dir, path.Join(rootDir, "packages")})
	if err != nil {
		return nil, fmt.Errorf("Failed to bind packages dir: %v", err.Error())
	}

	err = runCmd([]string{"/bin/mount", "--bind", sandboxDir, path.Join(rootDir, "host")})
	if err != nil {
		return nil, fmt.Errorf("Failed to bind host dir: %v", err.Error())
	}

	startCmd := []string{"/ol-init"}
	if indexHost != "" {
		startCmd = append(startCmd, indexHost)
	}
	if indexPort != "" {
		startCmd = append(startCmd, indexPort)
	}

	return NewOLContainerSandbox(sf.opts, rootDir, sandboxDir, id, startCmd, unshareFlags)
}

func (sf *OLContainerSBFactory) Cleanup() {
	for _, cgroup := range CGroupList {
		cgroupPath := path.Join("/sys/fs/cgroup", cgroup, OLCGroupName)
		os.Remove(cgroupPath)
	}
}

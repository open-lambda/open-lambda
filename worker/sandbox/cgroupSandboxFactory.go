package sandbox

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/open-lambda/open-lambda/worker/config"
)

// CgroupSBFactory is a SandboxFactory that creats docker sandboxes.
type CgroupSBFactory struct {
	opts *config.Config
}

// NewCgroupSBFactory creates a CgroupSBFactory.
func NewCgroupSBFactory(opts *config.Config) (*CgroupSBFactory, error) {
	return &CgroupSBFactory{opts: opts}, nil
}

func cmd(args []string) error {
	fmt.Printf("Execute: %s\n", strings.Join(args, " "))
	c := exec.Cmd{Path: args[0], Args: args}
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	return err
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (self *CgroupSBFactory) Create(handlerDir string, sandboxDir string) (Sandbox, error) {
	root, err := ioutil.TempDir(os.TempDir(), "sandbox_")
	if err != nil {
		return nil, err
	}

	err = cmd([]string{"/bin/mount", "--bind", "-o", "ro", self.opts.Cgroup_base, root})
	if err != nil {
		return nil, fmt.Errorf("Failed to bind base: %v", err.Error())
	}

	err = cmd([]string{"/bin/mount", "--bind", "-o", "ro", handlerDir, path.Join(root, "handler")})
	if err != nil {
		return nil, fmt.Errorf("Failed to bind handler dir: %v", err.Error())
	}

	err = cmd([]string{"/bin/mount", "--bind", sandboxDir, path.Join(root, "host")})
	if err != nil {
		return nil, fmt.Errorf("Failed to bind host dir: %v", err.Error())
	}

	sandbox, err := NewCgroupSandbox(self.opts, root)
	if err != nil {
		return nil, err
	}

	return sandbox, nil
}

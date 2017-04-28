/*

Provides the mechanism for managing a given Cgroup container-based lambda.

Must be paired with a CgroupSandboxManager which handles pulling handler
code, initializing containers, etc.

*/

package sandbox

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler/state"
)

type CgroupSandbox struct {
	opts     *config.Config
	root_dir string
	status   state.HandlerState
	nspid    int
}

func NewCgroupSandbox(opts *config.Config, root_dir string) (*CgroupSandbox, error) {
	sandbox := &CgroupSandbox{
		opts:     opts,
		root_dir: root_dir,
		status:   state.Stopped,
	}

	return sandbox, nil
}

func (s *CgroupSandbox) State() (hstate state.HandlerState, err error) {
	return s.status, nil
}

func (s *CgroupSandbox) Channel() (channel *SandboxChannel, err error) {
	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", filepath.Join(s.root_dir, "host", "ol.sock"))
	}
	tr := http.Transport{Dial: dial}

	// the server name doesn't matter since we have a sock file
	return &SandboxChannel{Url: "http://container/", Transport: tr}, nil
}

func (s *CgroupSandbox) Start() error {
	fmt.Printf("Start cgroup sandbox 1 %s\n", filepath.Join(s.root_dir, "handler"))

	cmd := []string{
		s.opts.Cgroup_init_path,
		s.root_dir,
		"/usr/bin/python",
		"/server.py",
	}
	env := []string{fmt.Sprintf("ol.config=%s", s.opts.SandboxConfJson())}
	attr := os.ProcAttr{
		Files: []*os.File{nil, os.Stdout, os.Stderr},
		Env:   env,
	}
	fmt.Printf("Use env=%v\n", attr)
	proc, err := os.StartProcess(cmd[0], cmd, &attr)
	if err != nil {
		return err
	}

	s.nspid = proc.Pid
	s.status = state.Running
	return nil
}

func (s *CgroupSandbox) Stop() error {
	// TODO(tyler)
	s.status = state.Stopped
	return nil
}

func (s *CgroupSandbox) Pause() error {
	// TODO(tyler)
	s.status = state.Paused
	return nil
}

func (s *CgroupSandbox) Unpause() error {
	// TODO(tyler)
	s.status = state.Running
	return nil
}

func (s *CgroupSandbox) Remove() error {
	// TODO(tyler)
	return nil
}

func (s *CgroupSandbox) Logs() (string, error) {
	// TODO(tyler)
	return "TODO", nil
}

func (s *CgroupSandbox) NSPid() int {
	return s.nspid
}

func (s *CgroupSandbox) ID() string {
	//TODO(tyler)
	return ""
}

func (s *CgroupSandbox) RunServer() error {
	//TODO(tyler)
	return nil
}

/*

Provides the mechanism for managing a given OLContainer container-based lambda.

Must be paired with a OLContainerSandboxManager which handles pulling handler
code, initializing containers, etc.

*/

package sandbox

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler/state"
)

type OLContainerSandbox struct {
	opts      *config.Config
	id        string
	rootDir   string
	indexHost string
	indexPort string
	status    state.HandlerState
	initProc  *os.Process
}

func NewOLContainerSandbox(opts *config.Config, rootDir, indexHost, indexPort, id string) (*OLContainerSandbox, error) {
	// create container cgroups
	for _, cgroup := range cgroupList {
		cgroupPath := path.Join("/sys/fs/cgroup/", cgroup, olCGroupName, id)
		if err := os.MkdirAll(cgroupPath, 0700); err != nil {
			return nil, err
		}
	}

	sandbox := &OLContainerSandbox{
		opts:      opts,
		id:        id,
		rootDir:   rootDir,
		indexHost: indexHost,
		indexPort: indexPort,
		status:    state.Stopped,
	}

	return sandbox, nil
}

func (s *OLContainerSandbox) State() (hstate state.HandlerState, err error) {
	return s.status, nil
}

func (s *OLContainerSandbox) Channel() (channel *SandboxChannel, err error) {
	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", filepath.Join(s.rootDir, "host", "ol.sock"))
	}
	tr := http.Transport{Dial: dial}

	// the server name doesn't matter since we have a sock file
	return &SandboxChannel{Url: "http://container/", Transport: tr}, nil
}

func (s *OLContainerSandbox) Start() error {
	initArgs := []string{s.rootDir, "/ol-init"}
	if s.indexHost != "" {
		initArgs = append(initArgs, s.indexHost)
	}
	if s.indexPort != "" {
		initArgs = append(initArgs, s.indexPort)
	}

	initCmd := exec.Command(
		s.opts.OLContainer_init_path,
		initArgs...,
	)

	initCmd.Env = []string{fmt.Sprintf("ol.config=%s", s.opts.SandboxConfJson())}
	if err := initCmd.Start(); err != nil {
		return err
	}

	s.initProc = initCmd.Process
	fmt.Printf("PID: %v\n", s.initProc.Pid)

	if err := s.CGroupEnter(strconv.Itoa(s.initProc.Pid)); err != nil {
		return err
	}

	s.status = state.Running
	return nil
}

func (s *OLContainerSandbox) Stop() error {
	// kill any remaining processes
	for _, cgroup := range cgroupList {
		procsPath := path.Join("/sys/fs/cgroup/", cgroup, olCGroupName, s.id, "cgroup.procs")
		pids, err := ioutil.ReadFile(procsPath)
		if err != nil {
			return err
		}

		for _, pidStr := range strings.Split(strings.TrimSpace(string(pids[:])), "\n") {
			if pidStr == "" {
				break
			}

			if pid, err := strconv.Atoi(pidStr); err != nil {
				return err
			} else if proc, err := os.FindProcess(pid); err != nil {
				return err
			} else if err := proc.Kill(); err != nil {
				return err
			}
		}
	}

	// avoid zombie python process
	if _, err := s.initProc.Wait(); err != nil {
		return err
	}

	s.status = state.Stopped
	return nil
}

func (s *OLContainerSandbox) Pause() error {
	freezerPath := path.Join("/sys/fs/cgroup/freezer", olCGroupName, s.id, "freezer.state")
	err := ioutil.WriteFile(freezerPath, []byte("FROZEN"), os.ModeAppend)
	if err != nil {
		return err
	}

	// TODO wait for it to freeze?

	s.status = state.Paused
	return nil
}

func (s *OLContainerSandbox) Unpause() error {
	freezerPath := path.Join("/sys/fs/cgroup/freezer", olCGroupName, s.id, "freezer.state")
	err := ioutil.WriteFile(freezerPath, []byte("THAWED"), os.ModeAppend)
	if err != nil {
		return err
	}

	// TODO wait for it to thaw?

	s.status = state.Running
	return nil
}

// TODO: continue cleanup attempt on error?
func (s *OLContainerSandbox) Remove() error {
	// remove cgroups
	for _, cgroup := range cgroupList {
		cgroupPath := path.Join("/sys/fs/cgroup/", cgroup, olCGroupName, s.id)
		if err := os.Remove(cgroupPath); err != nil {
			return err
		}
	}

	// unmount directories
	handler_dir := filepath.Join(s.rootDir, "handler")
	if err := syscall.Unmount(handler_dir, syscall.MNT_DETACH); err != nil {
		return err
	}

	host_dir := filepath.Join(s.rootDir, "host")
	if err := syscall.Unmount(host_dir, syscall.MNT_DETACH); err != nil {
		return err
	}

	pkgs_dir := filepath.Join(s.rootDir, "packages")
	if err := syscall.Unmount(pkgs_dir, syscall.MNT_DETACH); err != nil {
		return err
	}

	if err := syscall.Unmount(s.rootDir, syscall.MNT_DETACH); err != nil {
		return err
	}

	// remove everything
	return os.RemoveAll(s.rootDir)
}

func (s *OLContainerSandbox) Logs() (string, error) {
	// TODO(ed)
	return "TODO", nil
}

func (s *OLContainerSandbox) CGroupEnter(pid string) (err error) {
	// put process into each cgroup
	for _, cgroup := range cgroupList {
		tasksPath := path.Join("/sys/fs/cgroup/", cgroup, olCGroupName, s.id, "tasks")

		err := ioutil.WriteFile(tasksPath, []byte(pid), os.ModeAppend)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *OLContainerSandbox) NSPid() string {
	return strconv.Itoa(s.initProc.Pid)
}

func (s *OLContainerSandbox) ID() string {
	return s.id
}

func (s *OLContainerSandbox) RunServer() error {
	signal := exec.Command("kill", "-SIGUSR1", strconv.Itoa(s.initProc.Pid))
	if err := signal.Run(); err != nil {
		return err
	}

	return nil
}

func (s *OLContainerSandbox) MemoryCGroupPath() string {
	return fmt.Sprintf("/sys/fs/cgroup/memory/%s/%s/", olCGroupName, s.id)
}

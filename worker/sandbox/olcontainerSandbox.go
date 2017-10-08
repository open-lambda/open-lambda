/*

Provides the mechanism for managing a given OLContainer container-based lambda.

Must be paired with a OLContainerSandboxManager which handles pulling handler
code, initializing containers, etc.

*/

package sandbox

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler/state"
)

type OLContainerSandbox struct {
	opts         *config.Config
	id           string
	rootDir      string
	sandboxDir   string
	status       state.HandlerState
	initPid      string
	initCmd      *exec.Cmd
	startCmd     []string
	unshareFlags []string
}

func NewOLContainerSandbox(opts *config.Config, rootDir, sandboxDir, id string, startCmd, unshareFlags []string) (*OLContainerSandbox, error) {
	// create container cgroups
	for _, cgroup := range CGroupList {
		cgroupPath := path.Join("/sys/fs/cgroup/", cgroup, OLCGroupName, id)
		if err := os.MkdirAll(cgroupPath, 0700); err != nil {
			return nil, err
		}
	}

	sandbox := &OLContainerSandbox{
		opts:         opts,
		id:           id,
		rootDir:      rootDir,
		sandboxDir:   sandboxDir,
		unshareFlags: unshareFlags,
		status:       state.Stopped,
		startCmd:     startCmd,
	}

	return sandbox, nil
}

func (s *OLContainerSandbox) State() (hstate state.HandlerState, err error) {
	return s.status, nil
}

func (s *OLContainerSandbox) Channel() (channel *SandboxChannel, err error) {
	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", filepath.Join(s.sandboxDir, "ol.sock"))
	}
	tr := http.Transport{Dial: dial}

	// the server name doesn't matter since we have a sock file
	return &SandboxChannel{Url: "http://container/", Transport: tr}, nil
}

func (s *OLContainerSandbox) Start() error {
	initArgs := []string{s.opts.OLContainer_init_path, s.rootDir}
	initArgs = append(initArgs, s.startCmd...)
	initArgs = append(s.unshareFlags, initArgs...)

	s.initCmd = exec.Command(
		"unshare",
		initArgs...,
	)

	s.initCmd.Env = []string{fmt.Sprintf("ol.config=%s", s.opts.SandboxConfJson())}
	if err := s.initCmd.Start(); err != nil {
		return err
	}

	// wait up to 5s for server olcontainer_init to spawn
	start := time.Now()
	for {
		pgrep := exec.Command("pgrep", "-P", strconv.Itoa(s.initCmd.Process.Pid))
		out, err := pgrep.Output()
		if err == nil {
			s.initPid = strings.TrimSpace(string(out[:]))
			break
		}

		if time.Since(start).Seconds() > 5 {
			return fmt.Errorf("olcontainer_init failed to spawn after 5s")
		}
		time.Sleep(10 * time.Microsecond)
	}

	if err := s.CGroupEnter(s.initPid); err != nil {
		return err
	}

	s.status = state.Running
	return nil
}

func (s *OLContainerSandbox) Stop() error {
	// kill any remaining processes
	for _, cgroup := range CGroupList {
		procsPath := path.Join("/sys/fs/cgroup/", cgroup, OLCGroupName, s.id, "cgroup.procs")
		pids, err := ioutil.ReadFile(procsPath)
		if err != nil {
			return err
		}

		for _, pidStr := range strings.Split(strings.TrimSpace(string(pids[:])), "\n") {
			if pidStr == "" {
				break
			}

			// don't check errors because some might die before we get to them
			exec.Command("kill", "-9", pidStr).Run()
		}
	}

	// release unshare process resources
	s.initCmd.Wait()

	s.status = state.Stopped
	return nil
}

func (s *OLContainerSandbox) Pause() error {
	freezerPath := path.Join("/sys/fs/cgroup/freezer", OLCGroupName, s.id, "freezer.state")
	err := ioutil.WriteFile(freezerPath, []byte("FROZEN"), os.ModeAppend)
	if err != nil {
		return err
	}

	s.status = state.Paused
	return nil
}

func (s *OLContainerSandbox) Unpause() error {
	freezerPath := path.Join("/sys/fs/cgroup/freezer", OLCGroupName, s.id, "freezer.state")
	err := ioutil.WriteFile(freezerPath, []byte("THAWED"), os.ModeAppend)
	if err != nil {
		return err
	}

	s.status = state.Running
	return nil
}

func (s *OLContainerSandbox) Remove() error {
	// remove sockets if they exist
	if err := os.RemoveAll(filepath.Join(s.sandboxDir, "ol.sock")); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(s.sandboxDir, "fs.sock")); err != nil {
		return err
	}

	// remove cgroups
	for _, cgroup := range CGroupList {
		cgroupPath := path.Join("/sys/fs/cgroup/", cgroup, OLCGroupName, s.id)
		if err := os.Remove(cgroupPath); err != nil {
			return err
		}
	}

	// unmount directories
	handler_dir := filepath.Join(s.rootDir, "handler")
	if err := syscall.Unmount(handler_dir, syscall.MNT_DETACH); err != nil {
		log.Printf("failed to unmount handler dir: %s :: %s\n", handler_dir, err)
	}

	host_dir := filepath.Join(s.rootDir, "host")
	if err := syscall.Unmount(host_dir, syscall.MNT_DETACH); err != nil {
		log.Printf("failed to unmount host dir: %s :: %s\n", host_dir, err)
	}

	pkgs_dir := filepath.Join(s.rootDir, "packages")
	if err := syscall.Unmount(pkgs_dir, syscall.MNT_DETACH); err != nil {
		log.Printf("failed to unmount packages dir: %s :: %s\n", pkgs_dir, err)
	}

	if err := syscall.Unmount(s.rootDir, syscall.MNT_DETACH); err != nil {
		log.Printf("failed to unmount root dir: %s :: %s\n", s.rootDir, err)
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
	for _, cgroup := range CGroupList {
		tasksPath := path.Join("/sys/fs/cgroup/", cgroup, OLCGroupName, s.id, "tasks")

		err := ioutil.WriteFile(tasksPath, []byte(pid), os.ModeAppend)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *OLContainerSandbox) NSPid() string {
	return s.initPid
}

func (s *OLContainerSandbox) ID() string {
	return s.id
}

func (s *OLContainerSandbox) RunServer() error {
	signal := exec.Command("kill", "-SIGUSR1", s.initPid)
	if err := signal.Run(); err != nil {
		return err
	}

	return nil
}

func (s *OLContainerSandbox) MemoryCGroupPath() string {
	return fmt.Sprintf("/sys/fs/cgroup/memory/%s/%s/", OLCGroupName, s.id)
}

func (s *OLContainerSandbox) RootDir() string {
	return s.rootDir
}

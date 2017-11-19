/*

Provides the mechanism for managing a given SOCK container-based lambda.

Must be paired with a SOCKSandboxManager which handles pulling handler
code, initializing containers, etc.

*/

package sandbox

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler/state"
)

type SOCKSandbox struct {
	opts         *config.Config
	cgf          *CgroupFactory
	id           string
	cgId         string
	rootDir      string
	hostDir      string
	status       state.HandlerState
	initPid      string
	initCmd      *exec.Cmd
	startCmd     []string
	unshareFlags []string
	pipe         *os.File
}

func NewSOCKSandbox(cgf *CgroupFactory, opts *config.Config, rootDir, id string, startCmd, unshareFlags []string) (*SOCKSandbox, error) {
	// create container cgroups
	cgId, err := cgf.GetCg(id)
	if err != nil {
		return nil, err
	}

	sandbox := &SOCKSandbox{
		cgf:          cgf,
		opts:         opts,
		id:           id,
		cgId:         cgId,
		rootDir:      rootDir,
		unshareFlags: unshareFlags,
		status:       state.Stopped,
		startCmd:     startCmd,
	}

	return sandbox, nil
}

func (s *SOCKSandbox) State() (hstate state.HandlerState, err error) {
	return s.status, nil
}

func (s *SOCKSandbox) Channel() (channel *SandboxChannel, err error) {
	if s.hostDir == "" {
		return nil, fmt.Errorf("cannot call channel before calling mountDirs")
	}

	sockPath := filepath.Join(s.hostDir, "ol.sock")
	if len(sockPath) > 108 {
		return nil, fmt.Errorf("socket path length cannot exceed 108 characters (try moving cluster closer to the root directory")
	}

	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", sockPath)
	}
	tr := http.Transport{Dial: dial}

	// the server name doesn't matter since we have a sock file
	return &SandboxChannel{Url: "http://container/", Transport: tr}, nil
}

func (s *SOCKSandbox) Start() error {
	defer func(start time.Time) {
		if config.Timing {
			log.Printf("create container took %v\n", time.Since(start))
		}
	}(time.Now())

	initArgs := append(s.unshareFlags, s.rootDir)
	initArgs = append(initArgs, s.startCmd...)

	s.initCmd = exec.Command(
		"/usr/local/bin/sock-init",
		initArgs...,
	)

	s.initCmd.Env = []string{fmt.Sprintf("ol.config=%s", s.opts.SandboxConfJson())}

	// let the init program prints error to log, for debugging
	s.initCmd.Stderr = os.Stdout

	// setup the pipe
	pipeDir := filepath.Join(s.HostDir(), "init_pipe")
	pipe, err := os.OpenFile(pipeDir, os.O_RDWR, 0777)
	if err != nil {
		log.Fatalf("Cannot open pipe: %v\n", err)
	}
	s.pipe = pipe

	start := time.Now()
	if err := s.initCmd.Start(); err != nil {
		return err
	}

	ready := make(chan string, 1)
	defer close(ready)
	go func() {
		// message will be either 5 byte \0 padded pid (<65536), or "ready"
		pid := make([]byte, 6)
		n, err := s.Pipe().Read(pid[:5])
		if err != nil {
			log.Printf("Cannot read from stdout of sock: %v\n", err)
		} else if n != 5 {
			log.Printf("Expect to read 5 bytes, only %d read\n", n)
		} else {
			ready <- string(pid[:bytes.IndexByte(pid, 0)])
		}
	}()

	// wait up to 5s for server sock_init to spawn
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	select {
	case s.initPid = <-ready:
		if config.Timing {
			log.Printf("wait for sock_init took %v\n", time.Since(start))
		}
	case <-timeout.C:
		// clean up go routine
		if n, err := s.Pipe().Write([]byte("timeo")); err != nil {
			return err
		} else if n != 5 {
			return fmt.Errorf("Cannot write `timeo` to pipe\n")
		}
		return fmt.Errorf("sock_init failed to spawn after 5s")
	}

	if err := s.CGroupEnter(s.initPid); err != nil {
		return err
	}

	s.status = state.Running
	return nil
}

func (s *SOCKSandbox) Stop() error {
	if err := s.WaitForUnpause(5 * time.Second); err != nil {
		return err
	}

	start := time.Now()

	pid, _ := strconv.Atoi(s.initPid)
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find init process with pid=%d :: %v", pid, err)
	}
	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		log.Printf("failed to send kill signal to init process pid=%d :: %v", pid, err)
	}

	// let the initCmd (sock_init) to clean up all children
	_, err = s.initCmd.Process.Wait()
	if err != nil {
		log.Printf("failed to wait on initCmd pid=%d :: %v", s.initCmd.Process.Pid, err)
	}
	if config.Timing {
		log.Printf("kill processes took %v", time.Since(start))
	}

	s.status = state.Stopped
	return nil
}

func (s *SOCKSandbox) Pause() error {
	freezerPath := filepath.Join("/sys/fs/cgroup/freezer", OLCGroupName, s.cgId, "freezer.state")
	err := ioutil.WriteFile(freezerPath, []byte("FROZEN"), os.ModeAppend)
	if err != nil {
		return err
	}

	s.status = state.Paused
	return nil
}

func (s *SOCKSandbox) Unpause() error {
	statePath := filepath.Join("/sys/fs/cgroup/freezer", OLCGroupName, s.cgId, "freezer.state")

	err := ioutil.WriteFile(statePath, []byte("THAWED"), os.ModeAppend)
	if err != nil {
		return err
	}

	return nil
}

func (s *SOCKSandbox) WaitForUnpause(timeout time.Duration) error {
	// TODO: should we check parent_freezing to be sure?
	selfFreezingPath := filepath.Join("/sys/fs/cgroup/freezer", OLCGroupName, s.cgId, "freezer.self_freezing")

	start := time.Now()
	for time.Since(start) < timeout {
		freezerState, err := ioutil.ReadFile(selfFreezingPath)
		if err != nil {
			return fmt.Errorf("failed to check self_freezing state :: %v", err)
		}

		if strings.TrimSpace(string(freezerState[:])) == "0" {
			s.status = state.Running
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}

	return fmt.Errorf("sock didn't unpause after %v", timeout)
}

func (s *SOCKSandbox) Remove() error {
	if config.Timing {
		defer func(start time.Time) {
			log.Printf("remove took %v\n", time.Since(start))
		}(time.Now())
	}

	if err := syscall.Unmount(s.rootDir, syscall.MNT_DETACH); err != nil {
		log.Printf("unmount root dir %s failed :: %v\n", s.rootDir, err)
	}

	if err := os.RemoveAll(s.rootDir); err != nil {
		log.Printf("remove root dir %s failed :: %v\n", s.rootDir, err)
	}

	if err := os.RemoveAll(s.hostDir); err != nil {
		log.Printf("remove host dir %s failed :: %v\n", s.hostDir, err)
	}

	//TODO somehow wait for the processes to exit?
	time.Sleep(100 * time.Millisecond)
	// remove cgroups
	if err := s.cgf.PutCg(s.id, s.cgId); err != nil {
		log.Printf("Unable to delete cgroups: %v", err)
	}

	return nil
}

func (s *SOCKSandbox) Logs() (string, error) {
	// TODO(ed)
	return "TODO", nil
}

func (s *SOCKSandbox) CGroupEnter(pid string) (err error) {
	if pid == "" {
		return fmt.Errorf("empty pid passed to cgroupenter")
	}

	// put process into each cgroup
	for _, cgroup := range CGroupList {
		tasksPath := filepath.Join("/sys/fs/cgroup/", cgroup, OLCGroupName, s.cgId, "tasks")

		err := ioutil.WriteFile(tasksPath, []byte(pid), os.ModeAppend)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SOCKSandbox) NSPid() string {
	return s.initPid
}

func (s *SOCKSandbox) ID() string {
	return s.id
}

func (s *SOCKSandbox) RunServer() error {
	pid, err := strconv.Atoi(s.initPid)
	if err != nil {
		log.Printf("bad initPid string: %s :: %v", s.initPid, err)
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		log.Printf("failed to find initPid process with pid=%d :: %v", pid, err)
		return err
	}

	ready := make(chan bool, 1)
	defer close(ready)
	go func() {
		// wait for signal handler to be "ready"
		buf := make([]byte, 5)
		_, err = s.Pipe().Read(buf)
		if err != nil {
			log.Fatalf("Cannot read from stdout of sock: %v\n", err)
		} else if string(buf) != "ready" {
			log.Fatalf("In sockSandbox: Expect to see `ready` but sees %s\n", string(buf))
		}
		ready <- true
	}()

	// wait up to 5s for server sock_init to spawn
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	start := time.Now()
	select {
	case <-ready:
		if config.Timing {
			log.Printf("wait for init signal handler took %v\n", time.Since(start))
		}
	case <-timeout.C:
		if n, err := s.Pipe().Write([]byte("timeo")); err != nil {
			return err
		} else if n != 5 {
			return fmt.Errorf("Cannot write `timeo` to pipe\n")
		}
		return fmt.Errorf("sock_init failed to spawn after 5s")
	}

	err = proc.Signal(syscall.SIGUSR1)
	if err != nil {
		log.Printf("failed to send SIGUSR1 to pid=%d :: %v", pid, err)
		return err
	}

	return nil
}

func (s *SOCKSandbox) MemoryCGroupPath() string {
	return fmt.Sprintf("/sys/fs/cgroup/memory/%s/%s/", OLCGroupName, s.cgId)
}

func (s *SOCKSandbox) RootDir() string {
	return s.rootDir
}

func (s *SOCKSandbox) HostDir() string {
	return s.hostDir
}

func (s *SOCKSandbox) MountDirs(hostDir, handlerDir string) error {
	s.hostDir = hostDir

	pipDir := filepath.Join(hostDir, "pip")
	if err := os.Mkdir(pipDir, 0777); err != nil {
		return err
	}

	tmpDir := filepath.Join(hostDir, "tmp")
	if err := os.Mkdir(tmpDir, 0777); err != nil {
		return err
	}

	sbHostDir := filepath.Join(s.rootDir, "host")
	if err := syscall.Mount(hostDir, sbHostDir, "", BIND, ""); err != nil {
		return fmt.Errorf("failed to bind host dir: %v", err.Error())
	}

	sbTmpDir := filepath.Join(s.rootDir, "tmp")
	if err := syscall.Mount(tmpDir, sbTmpDir, "", BIND, ""); err != nil {
		return fmt.Errorf("failed to bind tmp dir: %v", err.Error())
	}

	if handlerDir != "" {
		sbHandlerDir := filepath.Join(s.rootDir, "handler")
		if err := syscall.Mount(handlerDir, sbHandlerDir, "", BIND, ""); err != nil {
			return fmt.Errorf("failed to bind handler dir: %s -> %s :: %v", handlerDir, sbHandlerDir, err.Error())
		} else if err := syscall.Mount("none", sbHandlerDir, "", BIND_RO, ""); err != nil {
			return fmt.Errorf("failed to bind handler dir RO: %v", err.Error())
		}
	}

	return nil
}

func (s *SOCKSandbox) Pipe() *os.File {
	return s.pipe
}

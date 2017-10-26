/*

Provides the mechanism for managing a given OLContainer container-based lambda.

Must be paired with a OLContainerSandboxManager which handles pulling handler
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

type OLContainerSandbox struct {
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
}

func NewOLContainerSandbox(cgf *CgroupFactory, opts *config.Config, rootDir, id string, startCmd, unshareFlags []string) (*OLContainerSandbox, error) {
	// create container cgroups
	cgId, err := cgf.GetCg(id)
	if err != nil {
		return nil, err
	}

	sandbox := &OLContainerSandbox{
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

func (s *OLContainerSandbox) State() (hstate state.HandlerState, err error) {
	return s.status, nil
}

func (s *OLContainerSandbox) Channel() (channel *SandboxChannel, err error) {
	if s.hostDir == "" {
		return nil, fmt.Errorf("cannot call channel before calling mountDirs")
	}

	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", filepath.Join(s.hostDir, "ol.sock"))
	}
	tr := http.Transport{Dial: dial}

	// the server name doesn't matter since we have a sock file
	return &SandboxChannel{Url: "http://container/", Transport: tr}, nil
}

func (s *OLContainerSandbox) Start() error {
	start := time.Now()
	defer func(start time.Time) {
		log.Printf("create container took %v\n", time.Since(start))
	}(start)

	initArgs := append(s.unshareFlags, s.rootDir)
	initArgs = append(initArgs, s.startCmd...)

	s.initCmd = exec.Command(
		s.opts.OLContainer_init_path,
		initArgs...,
	)

	s.initCmd.Env = []string{fmt.Sprintf("ol.config=%s", s.opts.SandboxConfJson())}

	// let the init program prints error to log, for debugging
	s.initCmd.Stderr = os.Stdout

	pipeDir := filepath.Join(s.HostDir(), "pipe")
	pipe, err := os.OpenFile(pipeDir, os.O_RDWR, 0777)
	if err != nil {
		log.Fatalf("Cannot open pipe: %v\n", err)
	}
	defer pipe.Close()

	cmdStart := time.Now()
	if err := s.initCmd.Start(); err != nil {
		return err
	}

	ready := make(chan string, 1)
	go func() {
		// message will be either 5 byte \0 padded pid (<65536), or "ready"
		pid := make([]byte, 6)
		n, err := pipe.Read(pid[:5])
		if err != nil {
			log.Fatalf("Cannot read from stdout of olcontainer: %v\n", err)
		} else if n != 5 {
			log.Fatalf("Expect to read 5 bytes, only %d read\n", n)
		}

		// TODO: make it less hacky
		if s.startCmd[0] == "/ol-init" {
			// wait for signal handler to be "ready"
			buf := make([]byte, 5)
			n, err = pipe.Read(buf)
			if err != nil {
				log.Fatalf("Cannot read from stdout of olcontainer: %v\n", err)
			} else if string(buf) != "ready" {
				log.Fatalf("In olcontainerSandbox: Expect to see `ready` but sees %s\n", string(buf))
			}
		}

		ready <- string(pid[:bytes.IndexByte(pid, 0)])
	}()

	// wait up to 5s for server olcontainer_init to spawn
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()

	select {
	case s.initPid = <-ready:
		log.Printf("wait for olcontainer_init took %v\n", time.Since(cmdStart))
	case <-timeout:
		return fmt.Errorf("olcontainer_init failed to spawn after 5s")
	}

	if err := s.CGroupEnter(s.initPid); err != nil {
		return err
	}

	s.status = state.Running
	return nil
}

func (s *OLContainerSandbox) Stop() error {
	if err := s.WaitForUnpause(5 * time.Second); err != nil {
		return err
	}

	start := time.Now()
	// kill any remaining processes
	procsPath := filepath.Join("/sys/fs/cgroup/memory", OLCGroupName, s.cgId, "cgroup.procs")
	pids, err := ioutil.ReadFile(procsPath)
	if err != nil {
		return err
	}

	for _, pidStr := range strings.Split(strings.TrimSpace(string(pids[:])), "\n") {
		if pidStr == "" {
			break
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			log.Printf("read bad pid string: %s :: %v", pidStr, err)
			continue
		}

		proc, err := os.FindProcess(pid)
		if err != nil {
			log.Printf("failed to find process with pid=%d :: %v", pid, err)
			continue
		}

		err = proc.Signal(syscall.SIGKILL)
		if err != nil {
			log.Printf("failed to send kill signal to pid=%d :: %v", pid, err)
		}
	}

	go func(s *OLContainerSandbox, start time.Time) {
		// release unshare process resources
		s.initCmd.Process.Kill()
		s.initCmd.Process.Wait()
		log.Printf("kill processes took %v", time.Since(start))
	}(s, start)

	s.status = state.Stopped
	return nil
}

func (s *OLContainerSandbox) Pause() error {
	freezerPath := filepath.Join("/sys/fs/cgroup/freezer", OLCGroupName, s.cgId, "freezer.state")
	err := ioutil.WriteFile(freezerPath, []byte("FROZEN"), os.ModeAppend)
	if err != nil {
		return err
	}

	s.status = state.Paused
	return nil
}

func (s *OLContainerSandbox) Unpause() error {
	statePath := filepath.Join("/sys/fs/cgroup/freezer", OLCGroupName, s.cgId, "freezer.state")

	err := ioutil.WriteFile(statePath, []byte("THAWED"), os.ModeAppend)
	if err != nil {
		return err
	}

	return nil
}

func (s *OLContainerSandbox) WaitForUnpause(timeout time.Duration) error {
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
		time.Sleep(100 * time.Microsecond)
	}

	return fmt.Errorf("olcontainer didn't unpause after %v", timeout)
}

func (s *OLContainerSandbox) Remove() error {
	start := time.Now()

	// remove cgroups
	if err := s.cgf.PutCg(s.id, s.cgId); err != nil {
		log.Printf("Unable to delete cgroups: %v", err)
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

	log.Printf("remove took %v\n", time.Since(start))

	return nil
}

func (s *OLContainerSandbox) Logs() (string, error) {
	// TODO(ed)
	return "TODO", nil
}

func (s *OLContainerSandbox) CGroupEnter(pid string) (err error) {
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

func (s *OLContainerSandbox) NSPid() string {
	return s.initPid
}

func (s *OLContainerSandbox) ID() string {
	return s.id
}

func (s *OLContainerSandbox) RunServer() error {
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

	err = proc.Signal(syscall.SIGURG)
	if err != nil {
		log.Printf("failed to send SIGUSR1 to pid=%d :: %v", pid, err)
		return err
	}

	return nil
}

func (s *OLContainerSandbox) MemoryCGroupPath() string {
	return fmt.Sprintf("/sys/fs/cgroup/memory/%s/%s/", OLCGroupName, s.cgId)
}

func (s *OLContainerSandbox) RootDir() string {
	return s.rootDir
}

func (s *OLContainerSandbox) HostDir() string {
	return s.hostDir
}

func (s *OLContainerSandbox) MountDirs(hostDir, handlerDir string) error {
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

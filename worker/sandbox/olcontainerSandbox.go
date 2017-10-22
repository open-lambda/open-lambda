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
	cgf          *CgroupFactory
	id           string
	cgId         string
	rootDir      string
	HostDir      string
	status       state.HandlerState
	initPid      string
	initCmd      *exec.Cmd
	startCmd     []string
	unshareFlags []string
}

func NewOLContainerSandbox(cgf *CgroupFactory, opts *config.Config, rootDir, id string, startCmd, unshareFlags []string) (*OLContainerSandbox, error) {
	// create container cgroups
	cgId := cgf.GetCg(id)

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
	if s.HostDir == "" {
		return nil, fmt.Errorf("cannot call channel before calling initHostDir")
	}

	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", filepath.Join(s.HostDir, "ol.sock"))
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
	log.Printf("wait for olcontainer_init took %v\n", time.Since(start))

	if err := s.CGroupEnter(s.initPid); err != nil {
		return err
	}

	s.status = state.Running
	return nil
}

func (s *OLContainerSandbox) Stop() error {
	start := time.Now()
	// kill any remaining processes
	procsPath := path.Join("/sys/fs/cgroup/memory", OLCGroupName, s.cgId, "cgroup.procs")
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
	}(s, start)

	s.status = state.Stopped
	return nil
}

func (s *OLContainerSandbox) Pause() error {
	freezerPath := path.Join("/sys/fs/cgroup/freezer", OLCGroupName, s.cgId, "freezer.state")
	err := ioutil.WriteFile(freezerPath, []byte("FROZEN"), os.ModeAppend)
	if err != nil {
		return err
	}

	s.status = state.Paused
	return nil
}

func (s *OLContainerSandbox) Unpause() error {
	freezerPath := path.Join("/sys/fs/cgroup/freezer", OLCGroupName, s.cgId, "freezer.state")
	err := ioutil.WriteFile(freezerPath, []byte("THAWED"), os.ModeAppend)
	if err != nil {
		return err
	}

	s.status = state.Running
	return nil
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

	if err := os.RemoveAll(s.HostDir); err != nil {
		log.Printf("remove host dir %s failed :: %v\n", s.HostDir, err)
	}

	log.Printf("remove took %v\n", time.Since(start))

	return nil
}

func (s *OLContainerSandbox) Logs() (string, error) {
	// TODO(ed)
	return "TODO", nil
}

func (s *OLContainerSandbox) CGroupEnter(pid string) (err error) {
	// put process into each cgroup
	for _, cgroup := range CGroupList {
		tasksPath := path.Join("/sys/fs/cgroup/", cgroup, OLCGroupName, s.cgId, "tasks")

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

func (s *OLContainerSandbox) mountDirs(hostDir, handlerDir string) error {
	s.HostDir = hostDir

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

	sbHandlerDir := filepath.Join(s.rootDir, "handler")
	if err := syscall.Mount(handlerDir, sbHandlerDir, "", BIND, ""); err != nil {
		return fmt.Errorf("failed to bind handler dir: %s -> %s :: %v", handlerDir, sbHandlerDir, err.Error())
	} else if err := syscall.Mount("none", sbHandlerDir, "", BIND_RO, ""); err != nil {
		return fmt.Errorf("failed to bind handler dir RO: %v", err.Error())
	}

	return nil
}

/*

Provides the mechanism for managing a given SOCK container-based lambda.

Must be paired with a SOCKContainerManager which handles pulling handler
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
	"github.com/open-lambda/open-lambda/worker/util"
)

type SOCKContainer struct {
	opts         *config.Config
	cgf          *CgroupFactory
	id           string
	cgId         string
	rootDir      string
	hostDir      string
	status       state.HandlerState
	initPid      string
	initCmd      *exec.Cmd
	unshareFlags string
	startCmd     []string
	pipe         *os.File
}

func NewSOCKContainer(cgf *CgroupFactory, opts *config.Config, rootDir, id, unshareFlags string, startCmd []string) (*SOCKContainer, error) {
	// create container cgroups
	cgId, err := cgf.GetCg(id)
	if err != nil {
		return nil, err
	}

	sandbox := &SOCKContainer{
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

func (c *SOCKContainer) State() (hstate state.HandlerState, err error) {
	return c.status, nil
}

func (c *SOCKContainer) Channel() (channel *Channel, err error) {
	if c.hostDir == "" {
		return nil, fmt.Errorf("cannot call channel before calling mountDirs")
	}

	sockPath := filepath.Join(c.hostDir, "ol.sock")
	if len(sockPath) > 108 {
		return nil, fmt.Errorf("socket path length cannot exceed 108 characters (try moving cluster closer to the root directory")
	}

	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", sockPath)
	}
	tr := http.Transport{Dial: dial}

	// the server name doesn't matter since we have a sock file
	return &Channel{Url: "http://container/", Transport: tr}, nil
}

func (c *SOCKContainer) Start() error {
	defer func(start time.Time) {
		if config.Timing {
			log.Printf("create container took %v\n", time.Since(start))
		}
	}(time.Now())

	initArgs := []string{c.unshareFlags, c.rootDir}
	initArgs = append(initArgs, c.startCmd...)

	c.initCmd = exec.Command(
		"/usr/local/bin/sock-init",
		initArgs...,
	)

	c.initCmd.Env = []string{fmt.Sprintf("ol.config=%s", c.opts.SandboxConfJson())}

	// let the init program prints error to log, for debugging
	c.initCmd.Stderr = os.Stdout

	// setup the pipe
	pipeDir := filepath.Join(c.HostDir(), "init_pipe")
	pipe, err := os.OpenFile(pipeDir, os.O_RDWR, 0777)
	if err != nil {
		log.Fatalf("Cannot open pipe: %v\n", err)
	}
	c.pipe = pipe

	start := time.Now()
	if err := c.initCmd.Start(); err != nil {
		return err
	}

	ready := make(chan string, 1)
	defer close(ready)
	go func() {
		// message will be either 5 byte \0 padded pid (<65536), or "ready"
		pid := make([]byte, 6)
		n, err := c.Pipe().Read(pid[:5])
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
	case c.initPid = <-ready:
		if config.Timing {
			log.Printf("wait for sock_init took %v\n", time.Since(start))
		}
	case <-timeout.C:
		// clean up go routine
		if n, err := c.Pipe().Write([]byte("timeo")); err != nil {
			return err
		} else if n != 5 {
			return fmt.Errorf("Cannot write `timeo` to pipe\n")
		}
		return fmt.Errorf("sock_init failed to spawn after 5s")
	}

	if err := c.CGroupEnter(c.initPid); err != nil {
		return err
	}

	c.status = state.Running
	return nil
}

func (c *SOCKContainer) Stop() error {
	start := time.Now()

	// If we're using the PID namespace, we can just kill the init process
	// and the OS will SIGKILL the rest. If not (i.e., for import cache
	// containers), we need to kill them all.
	if strings.Contains(c.unshareFlags, "p") {
		pid, _ := strconv.Atoi(c.initPid)
		proc, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("failed to find init process with pid=%d :: %v", pid, err)
		}
		err = proc.Signal(syscall.SIGTERM)
		if err != nil {
			log.Printf("failed to send kill signal to init process pid=%d :: %v", pid, err)
		}
	} else {
		// kill any remaining processes
		procsPath := filepath.Join("/sys/fs/cgroup/memory", OLCGroupName, c.cgId, "cgroup.procs")
		pids, err := ioutil.ReadFile(procsPath)
		if err != nil {
			return err
		}

		for _, pidStr := range strings.Split(strings.TrimSpace(string(pids[:])), "\n") {
			if pidStr == "" {
				break
			}

			if err := util.KillPIDStr(pidStr); err != nil {
				log.Printf("failed to kill pid %v, cleanup may fail :: %v", err)
			}
		}
	}

	// wait for the initCmd to clean up its children
	_, err := c.initCmd.Process.Wait()
	if err != nil {
		log.Printf("failed to wait on initCmd pid=%d :: %v", c.initCmd.Process.Pid, err)
	}
	if config.Timing {
		log.Printf("kill processes took %v", time.Since(start))
	}

	c.status = state.Stopped
	return nil
}

func (c *SOCKContainer) Pause() error {
	freezerPath := filepath.Join("/sys/fs/cgroup/freezer", OLCGroupName, c.cgId, "freezer.state")
	err := ioutil.WriteFile(freezerPath, []byte("FROZEN"), os.ModeAppend)
	if err != nil {
		return err
	}

	c.status = state.Paused
	return nil
}

func (c *SOCKContainer) Unpause() error {
	statePath := filepath.Join("/sys/fs/cgroup/freezer", OLCGroupName, c.cgId, "freezer.state")

	err := ioutil.WriteFile(statePath, []byte("THAWED"), os.ModeAppend)
	if err != nil {
		return err
	}

	return c.waitForUnpause(5 * time.Second)
}

func (c *SOCKContainer) waitForUnpause(timeout time.Duration) error {
	// TODO: should we check parent_freezing to be sure?
	selfFreezingPath := filepath.Join("/sys/fs/cgroup/freezer", OLCGroupName, c.cgId, "freezer.self_freezing")

	start := time.Now()
	for time.Since(start) < timeout {
		freezerState, err := ioutil.ReadFile(selfFreezingPath)
		if err != nil {
			return fmt.Errorf("failed to check self_freezing state :: %v", err)
		}

		if strings.TrimSpace(string(freezerState[:])) == "0" {
			c.status = state.Running
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}

	return fmt.Errorf("sock didn't unpause after %v", timeout)
}

func (c *SOCKContainer) Remove() error {
	if config.Timing {
		defer func(start time.Time) {
			log.Printf("remove took %v\n", time.Since(start))
		}(time.Now())
	}

	if err := syscall.Unmount(c.rootDir, syscall.MNT_DETACH); err != nil {
		log.Printf("unmount root dir %s failed :: %v\n", c.rootDir, err)
	}

	if err := os.RemoveAll(c.rootDir); err != nil {
		log.Printf("remove root dir %s failed :: %v\n", c.rootDir, err)
	}

	if err := os.RemoveAll(c.hostDir); err != nil {
		log.Printf("remove host dir %s failed :: %v\n", c.hostDir, err)
	}

	// remove cgroups
	if err := c.cgf.PutCg(c.id, c.cgId); err != nil {
		log.Printf("Unable to delete cgroups: %v", err)
	}

	return nil
}

func (c *SOCKContainer) Logs() (string, error) {
	// TODO(ed)
	return "TODO", nil
}

func (c *SOCKContainer) CGroupEnter(pid string) (err error) {
	if pid == "" {
		return fmt.Errorf("empty pid passed to cgroupenter")
	}

	// put process into each cgroup
	for _, cgroup := range CGroupList {
		tasksPath := filepath.Join("/sys/fs/cgroup/", cgroup, OLCGroupName, c.cgId, "tasks")

		err := ioutil.WriteFile(tasksPath, []byte(pid), os.ModeAppend)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *SOCKContainer) NSPid() string {
	return c.initPid
}

func (c *SOCKContainer) ID() string {
	return c.id
}

func (c *SOCKContainer) RunServer() error {
	pid, err := strconv.Atoi(c.initPid)
	if err != nil {
		log.Printf("bad initPid string: %s :: %v", c.initPid, err)
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
		_, err = c.Pipe().Read(buf)
		if err != nil {
			log.Fatalf("Cannot read from stdout of sock: %v\n", err)
		} else if string(buf) != "ready" {
			log.Fatalf("In sockContainer: Expect to see `ready` but sees %s\n", string(buf))
		}
		ready <- true
	}()

	// wait up to 5s for SOCK server to spawn
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	start := time.Now()
	select {
	case <-ready:
		if config.Timing {
			log.Printf("wait for init signal handler took %v\n", time.Since(start))
		}
	case <-timeout.C:
		if n, err := c.Pipe().Write([]byte("timeo")); err != nil {
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

func (c *SOCKContainer) MemoryCGroupPath() string {
	return fmt.Sprintf("/sys/fs/cgroup/memory/%s/%s/", OLCGroupName, c.cgId)
}

func (c *SOCKContainer) RootDir() string {
	return c.rootDir
}

func (c *SOCKContainer) HostDir() string {
	return c.hostDir
}

func (c *SOCKContainer) MountDirs(hostDir, handlerDir string) error {
	c.hostDir = hostDir

	tmpDir := filepath.Join(hostDir, "tmp")
	if err := os.Mkdir(tmpDir, 0777); err != nil {
		return err
	}

	sbHostDir := filepath.Join(c.rootDir, "host")
	if err := syscall.Mount(hostDir, sbHostDir, "", BIND, ""); err != nil {
		return fmt.Errorf("failed to bind host dir: %v", err.Error())
	}

	sbTmpDir := filepath.Join(c.rootDir, "tmp")
	if err := syscall.Mount(tmpDir, sbTmpDir, "", BIND, ""); err != nil {
		return fmt.Errorf("failed to bind tmp dir: %v", err.Error())
	}

	if handlerDir != "" {
		sbHandlerDir := filepath.Join(c.rootDir, "handler")
		if err := syscall.Mount(handlerDir, sbHandlerDir, "", BIND, ""); err != nil {
			return fmt.Errorf("failed to bind handler dir: %s -> %s :: %v", handlerDir, sbHandlerDir, err.Error())
		} else if err := syscall.Mount("none", sbHandlerDir, "", BIND_RO, ""); err != nil {
			return fmt.Errorf("failed to bind handler dir RO: %v", err.Error())
		}
	}

	return nil
}

func (c *SOCKContainer) Pipe() *os.File {
	return c.pipe
}

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

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/handler/state"
	"github.com/open-lambda/open-lambda/ol/util"
)

type SOCKContainer struct {
	opts             *config.Config
	cgf              *CgroupFactory
	id               string
	cgId             string
	containerRootDir string
	baseDir          string
	codeDir          string
	scratchDir       string
	status           state.HandlerState
	initPid          string
	initCmd          *exec.Cmd
	unshareFlags     string
	startCmd         []string
	pipe             *os.File
}

func NewSOCKContainer(
	id, containerRootDir, baseDir, codeDir, scratchDir string,
	cgf *CgroupFactory, opts *config.Config, unshareFlags string, startCmd []string) *SOCKContainer {

	return &SOCKContainer{
		id:               id,
		containerRootDir: containerRootDir,
		baseDir:          baseDir,
		codeDir:          codeDir,
		scratchDir:       scratchDir,
		cgf:              cgf,
		opts:             opts,
		unshareFlags:     unshareFlags,
		status:           state.Stopped,
		startCmd:         startCmd,
	}
}

func (c *SOCKContainer) State() (hstate state.HandlerState, err error) {
	return c.status, nil
}

func (c *SOCKContainer) Channel() (channel *Channel, err error) {
	sockPath := filepath.Join(c.scratchDir, "ol.sock")
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

func (c *SOCKContainer) Start() (err error) {
	defer func(start time.Time) {
		if config.Timing {
			log.Printf("create container took %v\n", time.Since(start))
		}
	}(time.Now())

	// FILE SYSTEM STEP 1: mount base
	if err := os.Mkdir(c.containerRootDir, 0777); err != nil {
		return err
	}

	if err := syscall.Mount(c.baseDir, c.containerRootDir, "", BIND, ""); err != nil {
		return fmt.Errorf("failed to bind root dir: %s -> %s :: %v\n", c.baseDir, c.containerRootDir, err)
	}

	if err := syscall.Mount("none", c.containerRootDir, "", BIND_RO, ""); err != nil {
		return fmt.Errorf("failed to bind root dir RO: %s :: %v\n", c.containerRootDir, err)
	}

	if err := syscall.Mount("none", c.containerRootDir, "", PRIVATE, ""); err != nil {
		return fmt.Errorf("failed to make root dir private :: %v", err)
	}

	// FILE SYSTEM STEP 2: code dir
	if c.codeDir != "" {
		sbCodeDir := filepath.Join(c.containerRootDir, "handler")

		if err := syscall.Mount(c.codeDir, sbCodeDir, "", BIND, ""); err != nil {
			return fmt.Errorf("failed to bind code dir: %s -> %s :: %v", c.codeDir, sbCodeDir, err.Error())
		}

		if err := syscall.Mount("none", sbCodeDir, "", BIND_RO, ""); err != nil {
			return fmt.Errorf("failed to bind code dir RO: %v", err.Error())
		}
	}

	// FILE SYSTEM STEP 3: scratch dir (tmp and communication)
	if err := os.MkdirAll(c.scratchDir, 0777); err != nil {
		return err
	}

	tmpDir := filepath.Join(c.scratchDir, "tmp")
	if err := os.Mkdir(tmpDir, 0777); err != nil {
		return err
	}

	sbScratchDir := filepath.Join(c.containerRootDir, "host")
	if err := syscall.Mount(c.scratchDir, sbScratchDir, "", BIND, ""); err != nil {
		return fmt.Errorf("failed to bind scratch dir: %v", err.Error())
	}

	sbTmpDir := filepath.Join(c.containerRootDir, "tmp")
	if err := syscall.Mount(tmpDir, sbTmpDir, "", BIND, ""); err != nil {
		return fmt.Errorf("failed to bind tmp dir: %v", err.Error())
	}

	pipe := filepath.Join(c.scratchDir, "init_pipe") // communicate with init process
	if err := syscall.Mkfifo(pipe, 0777); err != nil {
		return err
	}

	c.pipe, err = os.OpenFile(pipe, os.O_RDWR, 0777)
	if err != nil {
		return fmt.Errorf("Cannot open pipe: %v\n", err)
	}

	pipe = filepath.Join(c.scratchDir, "server_pipe") // communicate with lambda server
	if err := syscall.Mkfifo(pipe, 0777); err != nil {
		return err
	}

	// START INIT PROC (sets up namespaces)
	initArgs := []string{c.unshareFlags, c.containerRootDir}
	initArgs = append(initArgs, c.startCmd...)

	c.initCmd = exec.Command(
		"/usr/local/bin/sock-init",
		initArgs...,
	)

	c.initCmd.Env = []string{fmt.Sprintf("ol.config=%s", c.opts.SandboxConfJson())}
	c.initCmd.Stderr = os.Stdout // for debugging

	start := time.Now()
	if err := c.initCmd.Start(); err != nil {
		return err
	}

	// wait up to 5s for server sock_init to spawn
	ready := make(chan string, 1)
	defer close(ready)
	go func() {
		// message will be either 5 byte \0 padded pid (<65536), or "ready"
		pid := make([]byte, 6)
		n, err := c.pipe.Read(pid[:5])

		// TODO: return early

		if err != nil {
			log.Printf("Cannot read from stdout of sock: %v\n", err)
		} else if n != 5 {
			log.Printf("Expect to read 5 bytes, only %d read\n", n)
		} else {
			ready <- string(pid[:bytes.IndexByte(pid, 0)])
		}
	}()

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	select {
	case c.initPid = <-ready:
		if config.Timing {
			log.Printf("wait for sock_init took %v\n", time.Since(start))
		}
	case <-timeout.C:
		// clean up go routine
		if n, err := c.pipe.Write([]byte("timeo")); err != nil {
			return err
		} else if n != 5 {
			return fmt.Errorf("Cannot write `timeo` to pipe\n")
		}
		return fmt.Errorf("sock_init failed to spawn after 5s")
	}

	// JOIN A CGROUP
	cgId, err := c.cgf.GetCg(c.id)
	if err != nil {
		return err
	}
	c.cgId = cgId

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

	if err := syscall.Unmount(c.containerRootDir, syscall.MNT_DETACH); err != nil {
		log.Printf("unmount root dir %s failed :: %v\n", c.containerRootDir, err)
	}

	if err := os.RemoveAll(c.containerRootDir); err != nil {
		log.Printf("remove root dir %s failed :: %v\n", c.containerRootDir, err)
	}

	if err := os.RemoveAll(c.scratchDir); err != nil {
		log.Printf("remove host dir %s failed :: %v\n", c.scratchDir, err)
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
		_, err = c.pipe.Read(buf)
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
		if n, err := c.pipe.Write([]byte("timeo")); err != nil {
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
	return c.containerRootDir
}

func (c *SOCKContainer) HostDir() string {
	return c.scratchDir
}

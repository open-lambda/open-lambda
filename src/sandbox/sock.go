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
)

type SOCKContainer struct {
	cg               *Cgroup
	id               string
	containerRootDir string
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
	id, containerRootDir, codeDir, scratchDir string,
	cgPool *CgroupPool, unshareFlags string, startCmd []string,
	cacheMgr *CacheManager, imports []string) (sandbox *SOCKContainer, err error) {

	c := &SOCKContainer{
		id:               id,
		containerRootDir: containerRootDir,
		codeDir:          codeDir,
		scratchDir:       scratchDir,
		unshareFlags:     unshareFlags,
		status:           state.Stopped,
		startCmd:         startCmd,
	}

	if err := c.start(cgPool); err != nil {
		c.Destroy()
		return nil, err
	}

	if cacheMgr != nil {
		err = cacheMgr.Provision(c, imports)
	} else {
		err = c.runServer()
	}

	if err != nil {
		c.Destroy()
		return nil, err
	}

	return c, nil
}

// add ID to each log message so we know which logs correspond to
// which containers
func (c *SOCKContainer) printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s [SOCK %s]", strings.TrimRight(msg, "\n"), c.id)
}

func (c *SOCKContainer) ID() string {
	return c.id
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

func (c *SOCKContainer) start(cgPool *CgroupPool) (err error) {
	defer func(start time.Time) {
		if config.Conf.Timing {
			c.printf("create container took %v\n", time.Since(start))
		}
	}(time.Now())

	// FILE SYSTEM STEP 1: mount base
	if err := os.Mkdir(c.containerRootDir, 0777); err != nil {
		return err
	}

	baseDir := config.Conf.SOCK_base_path
	if err := syscall.Mount(baseDir, c.containerRootDir, "", BIND, ""); err != nil {
		return fmt.Errorf("failed to bind root dir: %s -> %s :: %v\n", baseDir, c.containerRootDir, err)
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

	c.initCmd.Env = []string{fmt.Sprintf("ol.config=%s", config.SandboxConfJson())}
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
			c.printf("Cannot read from stdout of sock: %v\n", err)
		} else if n != 5 {
			c.printf("Expect to read 5 bytes, only %d read\n", n)
		} else {
			ready <- string(pid[:bytes.IndexByte(pid, 0)])
		}
	}()

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	select {
	case c.initPid = <-ready:
		if config.Conf.Timing {
			c.printf("wait for sock_init took %v\n", time.Since(start))
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
	c.cg = cgPool.GetCg()
	if err := c.cg.AddPid(c.initPid); err != nil {
		return err
	}

	c.status = state.Running
	return nil
}

func (c *SOCKContainer) Pause() error {
	c.cg.Pause()
	c.status = state.Paused
	return nil
}

func (c *SOCKContainer) Unpause() error {
	err := c.cg.Unpause()
	c.status = state.Running
	return err
}

func (c *SOCKContainer) Destroy() {
	if err := c.destroy(); err != nil {
		c.printf("Failed to cleanup container %v: %v", c.id, err)
	}
}

func (c *SOCKContainer) destroy() error {
	c.printf("destroy\n")

	if config.Conf.Timing {
		defer func(start time.Time) {
			c.printf("remove took %v\n", time.Since(start))
		}(time.Now())
	}

	c.printf("Pause/KillAllProcs/Unpause\n")
	if err := c.cg.Pause(); err != nil {
		return err
	}
	if err := c.cg.KillAllProcs(); err != nil {
		return err
	}
	if err := c.cg.Unpause(); err != nil {
		return err
	}

	// wait for the initCmd to clean up its children
	c.printf("wait for init to die\n")
	_, err := c.initCmd.Process.Wait()
	if err != nil {
		c.printf("failed to wait on initCmd pid=%d :: %v", c.initCmd.Process.Pid, err)
	}

	c.printf("unmount and remove dirs\n")

	if err := syscall.Unmount(c.containerRootDir, syscall.MNT_DETACH); err != nil {
		c.printf("unmount root dir %s failed :: %v\n", c.containerRootDir, err)
	}

	if err := os.RemoveAll(c.containerRootDir); err != nil {
		c.printf("remove root dir %s failed :: %v\n", c.containerRootDir, err)
	}

	if err := os.RemoveAll(c.scratchDir); err != nil {
		c.printf("remove host dir %s failed :: %v\n", c.scratchDir, err)
	}

	c.cg.Release()

	return nil
}

func (c *SOCKContainer) Logs() (string, error) {
	// TODO(ed)
	return "TODO", nil
}

func (c *SOCKContainer) runServer() error {
	pid, err := strconv.Atoi(c.initPid)
	if err != nil {
		c.printf("bad initPid string: %s :: %v", c.initPid, err)
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		c.printf("failed to find initPid process with pid=%d :: %v", pid, err)
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
		if config.Conf.Timing {
			c.printf("wait for init signal handler took %v\n", time.Since(start))
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
		c.printf("failed to send SIGUSR1 to pid=%d :: %v", pid, err)
		return err
	}

	return nil
}

func (c *SOCKContainer) MemUsageKB() int {
	usagePath := c.cg.Path("memory", "memory.usage_in_bytes")
	buf, err := ioutil.ReadFile(usagePath)
	if err != nil {
		panic(fmt.Sprintf("get usage failed: %v", err))
	}

	str := strings.TrimSpace(string(buf[:]))
	usage, err := strconv.Atoi(str)
	if err != nil {
		panic(fmt.Sprintf("atoi failed: %v", err))
	}

	return usage / 1024
}

func (c *SOCKContainer) HostDir() string {
	return c.scratchDir
}

// fork a new process from the Zygote in src, relocate it to be the server in dst
func (src *SOCKContainer) Fork(dst *SOCKContainer, imports []string, handler bool) error {
	sockPath := fmt.Sprintf("%s/fs.sock", src.HostDir())
	targetPid := dst.initPid
	rootDir := dst.containerRootDir
	pid, err := forkRequest(sockPath, targetPid, rootDir, imports, handler)
	if err != nil {
		return err
	}

	if err = dst.cg.AddPid(pid); err != nil {
		return err
	}

	return nil
}

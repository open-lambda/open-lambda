/*

Provides the mechanism for managing a given SOCK container-based lambda.

Must be paired with a SOCKContainerManager which handles pulling handler
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
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/ol/config"
)

type SOCKContainer struct {
	cg               *Cgroup
	id               string
	containerRootDir string
	codeDir          string
	scratchDir       string
	hostInitCmd      *exec.Cmd
	guestInitPid     string
	initPipe         *os.File
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

func (c *SOCKContainer) Channel() (tr *http.Transport, err error) {
	// note, for debugging, you can directly contact the sock file like this:
	// curl -XPOST --unix-socket ./ol.sock http:/test -d '{"some": "data"}'

	sockPath := filepath.Join(c.scratchDir, "ol.sock")
	if len(sockPath) > 108 {
		return nil, fmt.Errorf("socket path length cannot exceed 108 characters (try moving cluster closer to the root directory")
	}

	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", sockPath)
	}
	return &http.Transport{Dial: dial}, nil
}

func (c *SOCKContainer) startSock2(bootPy []string) (err error) {
	// get FDs to cgroups
	cgFiles := make([]*os.File, len(cgroupList))
	for i, name := range cgroupList {
		path := c.cg.Path(name, "cgroup.procs")
		fd, err := syscall.Open(path, syscall.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		cgFiles[i] = os.NewFile(uintptr(fd), path)
		defer cgFiles[i].Close()
	}

	sbBootPath := filepath.Join("host", "bootstrap.py")
	bootPath := filepath.Join(c.containerRootDir, sbBootPath)
	code := []byte(strings.Join(bootPy, "\n"))
	if err := ioutil.WriteFile(bootPath, code, 0600); err != nil {
		return err
	}

	cmd := exec.Command(
		"chroot", c.containerRootDir, "python3", "sock2.py", sbBootPath, strconv.Itoa(len(cgFiles)),
	)
	cmd.ExtraFiles = cgFiles

	// TODO: route this to a file
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Start()
}

func (c *SOCKContainer) populateRoot() (err error) {
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

	return nil
}

func (c *SOCKContainer) Pause() (err error) {
	c.cg.Pause()
	return nil
}

func (c *SOCKContainer) Unpause() (err error) {
	return c.cg.Unpause()
}

func (c *SOCKContainer) Destroy() {
	if err := c.destroy(); err != nil {
		panic(fmt.Sprintf("failed to destroy SOCK sandbox: %v", err))
	}
}

func (c *SOCKContainer) destroy() error {
	if config.Conf.Timing {
		defer func(start time.Time) {
			c.printf("remove took %v\n", time.Since(start))
		}(time.Now())
	}

	// kill all procs INSIDE the cgroup
	if c.cg != nil {
		c.printf("kill all procs in CG\n")
		if err := c.cg.KillAllProcs(); err != nil {
			return err
		}

		c.cg.Release()
	}

	// kill the host init process OUTSIDE the cgroup
	if c.hostInitCmd != nil {
		if err := c.hostInitCmd.Process.Kill(); err != nil {
			return err
		}

		c.printf("wait for host init process (PID %d) to die\n", c.hostInitCmd.Process.Pid)
		_, err := c.hostInitCmd.Process.Wait()
		if err != nil {
			c.printf("failed to wait on hostInitCmd pid=%d :: %v", c.hostInitCmd.Process.Pid, err)
		}
	}

	c.printf("unmount and remove dirs\n")

	if err := syscall.Unmount(c.containerRootDir, syscall.MNT_DETACH); err != nil {
		c.printf("unmount root dir %s failed :: %v\n", c.containerRootDir, err)
	}

	if err := os.RemoveAll(c.containerRootDir); err != nil {
		c.printf("remove root dir %s failed :: %v\n", c.containerRootDir, err)
	}

	// TODO: find balance between cleaning up, and preserving this for debug
	// if err := os.RemoveAll(c.scratchDir); err != nil {
	//     c.printf("remove host dir %s failed :: %v\n", c.scratchDir, err)
	//}

	return nil
}

func (c *SOCKContainer) MemUsageKB() (kb int, err error) {
	usagePath := c.cg.Path("memory", "memory.usage_in_bytes")
	buf, err := ioutil.ReadFile(usagePath)
	if err != nil {
		fmt.Errorf("get usage failed: %v", err)
	}

	str := strings.TrimSpace(string(buf[:]))
	usage, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("atoi failed: %v", err)
	}

	return usage / 1024, nil
}

func (c *SOCKContainer) HostDir() string {
	return c.scratchDir
}

func (c *SOCKContainer) DebugString() string {
	var s string = fmt.Sprintf("SOCK %s\n", c.ID())

	s += fmt.Sprintf("HOST DIR: %s\n", c.HostDir())

	if pids, err := c.cg.GetPIDs(); err == nil {
		s += fmt.Sprintf("CGROUP PIDS: %s\n", strings.Join(pids, ", "))
	} else {
		s += fmt.Sprintf("CGROUP PIDS: unknown (%s)\n", err)
	}

	s += fmt.Sprintf("GUEST INIT PID: %s\n", c.guestInitPid)

	if c.hostInitCmd != nil && c.hostInitCmd.Process != nil {
		s += fmt.Sprintf("HOST INIT PID: %d\n", c.hostInitCmd.Process.Pid)
	} else {
		s += fmt.Sprintf("HOST INIT PID: unknown\n")
	}

	s += fmt.Sprintf("CGROUPS: %s\n", c.cg.Path("<RESOURCE>", ""))

	if state, err := ioutil.ReadFile(c.cg.Path("freezer", "freezer.state")); err == nil {
		s += fmt.Sprintf("FREEZE STATE: %s", state)
	} else {
		s += fmt.Sprintf("FREEZE STATE: unknown (%s)\n", err)
	}

	if kb, err := c.MemUsageKB(); err == nil {
		s += fmt.Sprintf("MEMORY USED: %.3fMB\n", float64(kb)/1024.0)
	} else {
		s += fmt.Sprintf("MEMORY USED: unknown (%s)\n", err)
	}

	return s
}

// fork a new process from the Zygote in c, relocate it to be the server in dst
func (c *SOCKContainer) fork(dst Sandbox, imports []string, isLeaf bool) (err error) {
	dstSock := dst.(*SOCKContainer)

	targetPid := dstSock.guestInitPid
	rootDir := dstSock.containerRootDir
	pid, err := c.forkRequest(targetPid, rootDir, imports, isLeaf)
	if err != nil {
		return err
	}

	c.printf("add forked PID %s to CG", pid)
	if err = dstSock.cg.AddPid(pid); err != nil {
		return err
	}

	return nil
}

// this forks init proc, then does execve to start server.py
func (c *SOCKContainer) runServer() error {
	pid, err := strconv.Atoi(c.guestInitPid)
	if err != nil {
		c.printf("bad guestInitPid string: %s :: %v", c.guestInitPid, err)
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		c.printf("failed to find guest init process with pid=%d :: %v", pid, err)
		return err
	}

	ready := make(chan bool, 1)
	defer close(ready)
	go func() {
		// wait for signal handler to be "ready"
		buf := make([]byte, 5)
		_, err = c.initPipe.Read(buf)
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
		if n, err := c.initPipe.Write([]byte("timeo")); err != nil {
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

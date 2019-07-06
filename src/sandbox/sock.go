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
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/stats"
)

type SOCKContainer struct {
	pool             *SOCKPool
	id               string
	containerRootDir string
	codeDir          string
	scratchDir       string
	cg               *Cgroup
	children         []Sandbox
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

func (c *SOCKContainer) HttpProxy() (p *httputil.ReverseProxy, err error) {
	// note, for debugging, you can directly contact the sock file like this:
	// curl -XPOST --unix-socket ./ol.sock http:/test -d '{"some": "data"}'

	sockPath := filepath.Join(c.scratchDir, "ol.sock")
	if len(sockPath) > 108 {
		return nil, fmt.Errorf("socket path length cannot exceed 108 characters (try moving cluster closer to the root directory")
	}

	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", sockPath)
	}

	tr := &http.Transport{Dial: dial}
	u, err := url.Parse("http://sock-container")
	if err != nil {
		panic(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	proxy.Transport = tr
	return proxy, nil
}

func (c *SOCKContainer) writeBootstrapCode(bootPy []string) (err error) {
	path := filepath.Join(c.containerRootDir, "host", "bootstrap.py")
	code := []byte(strings.Join(bootPy, "\n"))
	if err := ioutil.WriteFile(path, code, 0600); err != nil {
		return err
	}
	return nil
}

func (c *SOCKContainer) freshProc() (err error) {
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

	cmd := exec.Command(
		"chroot", c.containerRootDir, "python3", "-u",
		"sock2.py", "/host/bootstrap.py", strconv.Itoa(len(cgFiles)),
	)
	cmd.ExtraFiles = cgFiles

	// TODO: route this to a file
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	// sock2.py forks off a process in a new container, so this
	// won't block long
	return cmd.Wait()
}

func (c *SOCKContainer) populateRoot() (err error) {
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
	tmpDir := filepath.Join(c.scratchDir, "tmp")
	if err := os.Mkdir(tmpDir, 0777); err != nil {
		return err
	}

	sbScratchDir := filepath.Join(c.containerRootDir, "host")
	if err := syscall.Mount(c.scratchDir, sbScratchDir, "", BIND, ""); err != nil {
		return fmt.Errorf("failed to bind scratch dir: %v", err.Error())
	}

	// TODO: cheaper to handle with symlink in lambda image?
	sbTmpDir := filepath.Join(c.containerRootDir, "tmp")
	if err := syscall.Mount(tmpDir, sbTmpDir, "", BIND, ""); err != nil {
		return fmt.Errorf("failed to bind tmp dir: %v", err.Error())
	}

	return nil
}

func (c *SOCKContainer) Pause() (err error) {
	if err := c.cg.Pause(); err != nil {
		return err
	}

	// drop mem limit to what is used when we're paused, because
	// we know the Sandbox cannot allocate more when it's not
	// schedulable.  Then release saved memory back to the pool.
	oldLimit := c.cg.getMemLimitMB()
	newLimit := c.cg.getMemUsageMB() + 1
	if newLimit < oldLimit {
		c.cg.setMemLimitMB(newLimit)
		c.pool.mem.adjustAvailableMB(oldLimit - newLimit)
	}
	return nil
}

func (c *SOCKContainer) Unpause() (err error) {
	// block until we have enough mem to upsize limit to the
	// normal size before unpausing
	oldLimit := c.cg.getMemLimitMB()
	newLimit := config.Conf.Limits.Mem_mb
	c.pool.mem.adjustAvailableMB(oldLimit - newLimit)
	c.cg.setMemLimitMB(newLimit)

	return c.cg.Unpause()
}

func (c *SOCKContainer) Destroy() {
	if err := c.destroy(); err != nil {
		panic(fmt.Sprintf("failed to destroy SOCK sandbox: %v", err))
	}
}

func (c *SOCKContainer) destroy() error {
	// Destroy is recursive, so make sure all children are dead
	// first.  This is for memory accounting purposes (otherwise,
	// nobody is to blame for the memory allocated by the parent
	// before the children were forked).
	t := stats.T0("Destroy()/recursive-kill")
	for _, child := range c.children {
		child.Destroy()
	}
	t.T1()

	// kill all procs INSIDE the cgroup
	t = stats.T0("Destroy()/kill-procs")
	if c.cg != nil {
		c.printf("kill all procs in CG\n")
		if err := c.cg.KillAllProcs(); err != nil {
			return err
		}

		c.cg.Release()
	}
	t.T1()

	c.printf("unmount and remove dirs\n")
	t = stats.T0("Destroy()/detach-root")
	if err := syscall.Unmount(c.containerRootDir, syscall.MNT_DETACH); err != nil {
		c.printf("unmount root dir %s failed :: %v\n", c.containerRootDir, err)
	}
	t.T1()

	t = stats.T0("Destroy()/remove-root")
	if err := os.RemoveAll(c.containerRootDir); err != nil {
		c.printf("remove root dir %s failed :: %v\n", c.containerRootDir, err)
	}
	t.T1()

	// release memory used for this container
	c.pool.mem.adjustAvailableMB(c.cg.getMemLimitMB())

	return nil
}

func (c *SOCKContainer) DebugString() string {
	var s string = fmt.Sprintf("SOCK %s\n", c.ID())

	s += fmt.Sprintf("ROOT DIR: %s\n", c.containerRootDir)

	s += fmt.Sprintf("HOST DIR: %s\n", c.scratchDir)

	if pids, err := c.cg.GetPIDs(); err == nil {
		s += fmt.Sprintf("CGROUP PIDS: %s\n", strings.Join(pids, ", "))
	} else {
		s += fmt.Sprintf("CGROUP PIDS: unknown (%s)\n", err)
	}

	s += fmt.Sprintf("CGROUPS: %s\n", c.cg.Path("<RESOURCE>", ""))

	if state, err := ioutil.ReadFile(c.cg.Path("freezer", "freezer.state")); err == nil {
		s += fmt.Sprintf("FREEZE STATE: %s", state)
	} else {
		s += fmt.Sprintf("FREEZE STATE: unknown (%s)\n", err)
	}

	s += fmt.Sprintf("MEMORY USED: TODO (ask cgroup)\n")

	return s
}

// fork a new process from the Zygote in c, relocate it to be the server in dst
func (c *SOCKContainer) fork(dst Sandbox) (err error) {
	// TODO: find a better way to signal that the parent in
	// inoperational, and find a remedy
	spareMB := c.cg.getMemLimitMB() - c.cg.getMemUsageMB()
	if spareMB < 3 {
		return fmt.Errorf("only %vMB of spare memory in parent, rejecting fork request (need at least 3MB)", spareMB)
	}

	dstSock := dst.(*safeSandbox).Sandbox.(*SOCKContainer)

	origPids, err := c.cg.GetPIDs()
	if err != nil {
		return err
	}

	root, err := os.Open(dstSock.containerRootDir)
	if err != nil {
		return err
	}
	defer root.Close()

	cg := dstSock.cg
	memCG, err := os.OpenFile(cg.Path("memory", "cgroup.procs"), os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer memCG.Close()

	t := stats.T0("forkRequest")
	err = c.forkRequest(fmt.Sprintf("%s/ol.sock", c.scratchDir), root, memCG)
	if err != nil {
		return err
	}
	t.T1()

	// move new PIDs to new cgroup.
	//
	// Make multiple passes in case new processes are being
	// spawned (TODO: better way to do this?  This lets a forking
	// process potentially kill our cache entry, which isn't
	// great).
	t = stats.T0("move-to-cg-after-fork")
	for {
		currPids, err := c.cg.GetPIDs()
		if err != nil {
			return err
		}

		moved := 0

		for _, pid := range currPids {
			isOrig := false
			for _, origPid := range origPids {
				if pid == origPid {
					isOrig = true
					break
				}
			}
			if !isOrig {
				if err = dstSock.cg.AddPid(pid); err != nil {
					return err
				}
				moved += 1
			}
		}

		if moved == 0 {
			break
		}
	}

	c.children = append(c.children, dst)
	t.T1()

	return nil
}

func (c *SOCKContainer) Status(key SandboxStatus) (string, error) {
	switch key {
	case StatusMemFailures:
		return strconv.FormatBool(c.cg.ReadInt("memory", "memory.failcnt") > 0), nil
	default:
		return "", STATUS_UNSUPPORTED
	}
}

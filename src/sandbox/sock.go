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

	"github.com/open-lambda/open-lambda/ol/common"
)

type SOCKContainer struct {
	pool             *SOCKPool
	id               string
	meta             *SandboxMeta
	containerRootDir string
	codeDir          string
	scratchDir       string
	cg               *Cgroup

	// 1 for self, plus 1 for each child (we can't release memory
	// until all descendents are dead, because they share the
	// pages of this Container, but this is the only container
	// charged
	cgRefCount int

	parent   Sandbox
	children map[string]Sandbox
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

func (c *SOCKContainer) freshProc() (err error) {
	// get FDs to cgroups
	cgFiles := make([]*os.File, len(cgroupList))
	for i, name := range cgroupList {
		path := c.cg.Path(name, "tasks")
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
	cmd.Env = []string{} // for security, DO NOT expose host env to guest
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
	baseDir := common.Conf.SOCK_base_path
	if err := syscall.Mount(baseDir, c.containerRootDir, "", common.BIND, ""); err != nil {
		return fmt.Errorf("failed to bind root dir: %s -> %s :: %v\n", baseDir, c.containerRootDir, err)
	}

	if err := syscall.Mount("none", c.containerRootDir, "", common.BIND_RO, ""); err != nil {
		return fmt.Errorf("failed to bind root dir RO: %s :: %v\n", c.containerRootDir, err)
	}

	if err := syscall.Mount("none", c.containerRootDir, "", common.PRIVATE, ""); err != nil {
		return fmt.Errorf("failed to make root dir private :: %v", err)
	}

	// FILE SYSTEM STEP 2: code dir
	if c.codeDir != "" {
		sbCodeDir := filepath.Join(c.containerRootDir, "handler")

		if err := syscall.Mount(c.codeDir, sbCodeDir, "", common.BIND, ""); err != nil {
			return fmt.Errorf("failed to bind code dir: %s -> %s :: %v", c.codeDir, sbCodeDir, err.Error())
		}

		if err := syscall.Mount("none", sbCodeDir, "", common.BIND_RO, ""); err != nil {
			return fmt.Errorf("failed to bind code dir RO: %v", err.Error())
		}
	}

	// FILE SYSTEM STEP 3: scratch dir (tmp and communication)
	tmpDir := filepath.Join(c.scratchDir, "tmp")
	if err := os.Mkdir(tmpDir, 0777); err != nil && !os.IsExist(err) {
		return err
	}

	sbScratchDir := filepath.Join(c.containerRootDir, "host")
	if err := syscall.Mount(c.scratchDir, sbScratchDir, "", common.BIND, ""); err != nil {
		return fmt.Errorf("failed to bind scratch dir: %v", err.Error())
	}

	// TODO: cheaper to handle with symlink in lambda image?
	sbTmpDir := filepath.Join(c.containerRootDir, "tmp")
	if err := syscall.Mount(tmpDir, sbTmpDir, "", common.BIND, ""); err != nil {
		return fmt.Errorf("failed to bind tmp dir: %v", err.Error())
	}

	return nil
}

func (c *SOCKContainer) Pause() (err error) {
	if err := c.cg.Pause(); err != nil {
		return err
	}

	if common.Conf.Features.Downsize_paused_mem {
		// drop mem limit to what is used when we're paused, because
		// we know the Sandbox cannot allocate more when it's not
		// schedulable.  Then release saved memory back to the pool.
		oldLimit := c.cg.getMemLimitMB()
		newLimit := c.cg.getMemUsageMB() + 1
		if newLimit < oldLimit {
			c.cg.setMemLimitMB(newLimit)
			c.pool.mem.adjustAvailableMB(oldLimit - newLimit)
		}
	}
	return nil
}

func (c *SOCKContainer) Unpause() (err error) {
	if common.Conf.Features.Downsize_paused_mem {
		// block until we have enough mem to upsize limit to the
		// normal size before unpausing
		oldLimit := c.cg.getMemLimitMB()
		newLimit := common.Conf.Limits.Mem_mb
		c.pool.mem.adjustAvailableMB(oldLimit - newLimit)
		c.cg.setMemLimitMB(newLimit)
	}

	return c.cg.Unpause()
}

func (c *SOCKContainer) Destroy() {
	if err := c.cg.Pause(); err != nil {
		panic(err)
	}

	c.decCgRefCount()
}

// when the count goes to zero, it means (a) this container and (b)
// all it's descendents are destroyed. Thus, it's safe to release it's
// cgroups, and return the memory allocation to the memPool
func (c *SOCKContainer) decCgRefCount() {
	c.cgRefCount -= 1
	c.printf("CG ref count decremented to %d", c.cgRefCount)
	if c.cgRefCount < 0 {
		panic("cgRefCount should not be able to go negative")
	}

	// release all resources when we have no more dependents...
	if c.cgRefCount == 0 {
		t := common.T0("Destroy()/kill-procs")
		if c.cg != nil {
			pids := c.cg.KillAllProcs()
			c.printf("killed PIDs %v in CG\n", pids)
		}
		t.T1()

		c.printf("unmount and remove dirs\n")
		t = common.T0("Destroy()/detach-root")
		if err := syscall.Unmount(c.containerRootDir, syscall.MNT_DETACH); err != nil {
			c.printf("unmount root dir %s failed :: %v\n", c.containerRootDir, err)
		}
		t.T1()

		t = common.T0("Destroy()/remove-root")
		if err := os.RemoveAll(c.containerRootDir); err != nil {
			c.printf("remove root dir %s failed :: %v\n", c.containerRootDir, err)
		}
		t.T1()

		c.cg.Release()
		c.pool.mem.adjustAvailableMB(c.cg.getMemLimitMB())

		if c.parent != nil {
			c.parent.childExit(c)
		}
	}
}

func (c *SOCKContainer) childExit(child Sandbox) {
	delete(c.children, child.ID())
	c.decCgRefCount()
}

// fork a new process from the Zygote in c, relocate it to be the server in dst
func (c *SOCKContainer) fork(dst Sandbox) (err error) {
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
	memCG, err := os.OpenFile(cg.Path("memory", "tasks"), os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer memCG.Close()

	t := common.T0("forkRequest")
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
	t = common.T0("move-to-cg-after-fork")
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
				c.printf("move PID %v from CG %v to CG %v\n", pid, c.cg.Name, dstSock.cg.Name)
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
	t.T1()

	c.children[dst.ID()] = dst
	c.cgRefCount += 1
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

func (c *SOCKContainer) Meta() *SandboxMeta {
	return c.meta
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

	s += fmt.Sprintf("MEMORY USED: %d of %d MB\n",
		c.cg.getMemUsageMB(), c.cg.getMemLimitMB())

	s += fmt.Sprintf("MEMORY FAILURES: %d\n",
		c.cg.ReadInt("memory", "memory.failcnt"))

	return s
}

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
	rt_type          common.RuntimeType

	// 1 for self, plus 1 for each child (we can't release memory
	// until all descendants are dead, because they share the
	// pages of this Container, but this is the only container
	// charged)
	cgRefCount int

	parent   Sandbox
	children map[string]Sandbox
}

// add ID to each log message so we know which logs correspond to
// which containers
func (container *SOCKContainer) printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s [SOCK %s]", strings.TrimRight(msg, "\n"), container.id)
}

func (container *SOCKContainer) ID() string {
	return container.id
}

func (c *SOCKContainer) GetRuntimeType() common.RuntimeType {
	return c.rt_type
}

func (container *SOCKContainer) HttpProxy() (p *httputil.ReverseProxy, err error) {
	// note, for debugging, you can directly contact the sock file like this:
	// curl -XPOST --unix-socket ./ol.sock http:/test -d '{"some": "data"}'

	sockPath := filepath.Join(container.scratchDir, "ol.sock")
	if len(sockPath) > 108 {
		return nil, fmt.Errorf("socket path length cannot exceed 108 characters (try moving cluster closer to the root directory")
	}

	log.Printf("Connecting to container at '%s'", sockPath)

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

func (container *SOCKContainer) freshProc() (err error) {
	// get FDs to cgroups
	cgFiles := make([]*os.File, len(cgroupList))
	for i, name := range cgroupList {
		path := container.cg.Path(name, "tasks")
		fd, err := syscall.Open(path, syscall.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		cgFiles[i] = os.NewFile(uintptr(fd), path)
		defer cgFiles[i].Close()
	}

	var cmd *exec.Cmd

	if container.rt_type == common.RT_PYTHON {
		cmd = exec.Command(
			"chroot", container.containerRootDir, "python3", "-u",
			"/runtimes/python/server.py", "/host/bootstrap.py", strconv.Itoa(len(cgFiles)),
		)
	} else if container.rt_type == common.RT_BINARY {
        // Launch db proxy
        out, err := exec.Command("env", "RUST_LOG=debug", "RUST_BACKTRACE=full",
            "./ol-database-proxy", container.scratchDir,).CombinedOutput()

        if err == nil {
            log.Print("Started database proxy")
        } else {
            return fmt.Errorf("Failed to start database proxy. Output was:\n%s", out,)
        }

		cmd = exec.Command(
			"chroot", container.containerRootDir,
			"env", "RUST_BACKTRACE=full", "/runtimes/rust/server", strconv.Itoa(len(cgFiles)),
		)
	} else {
		return fmt.Errorf("Unsupported runtime")
	}

	cmd.Env = []string{} // for security, DO NOT expose host env to guest
	cmd.ExtraFiles = cgFiles

	// TODO: route this to a file
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	// Runtimes will fork off anew container,
    // so this won't block long
	return cmd.Wait()
}

func (container *SOCKContainer) populateRoot() (err error) {
	// FILE SYSTEM STEP 1: mount base
	baseDir := common.Conf.SOCK_base_path
	if err := syscall.Mount(baseDir, container.containerRootDir, "", common.BIND, ""); err != nil {
		return fmt.Errorf("failed to bind root dir: %s -> %s :: %v\n", baseDir, container.containerRootDir, err)
	}

	if err := syscall.Mount("none", container.containerRootDir, "", common.BIND_RO, ""); err != nil {
		return fmt.Errorf("failed to bind root dir RO: %s :: %v\n", container.containerRootDir, err)
	}

	if err := syscall.Mount("none", container.containerRootDir, "", common.PRIVATE, ""); err != nil {
		return fmt.Errorf("failed to make root dir private :: %v", err)
	}

	// FILE SYSTEM STEP 2: code dir
	if container.codeDir != "" {
		sbCodeDir := filepath.Join(container.containerRootDir, "handler")

		if err := syscall.Mount(container.codeDir, sbCodeDir, "", common.BIND, ""); err != nil {
			return fmt.Errorf("failed to bind code dir: %s -> %s :: %v", container.codeDir, sbCodeDir, err.Error())
		}

		if err := syscall.Mount("none", sbCodeDir, "", common.BIND_RO, ""); err != nil {
			return fmt.Errorf("failed to bind code dir RO: %v", err.Error())
		}
	}

	// FILE SYSTEM STEP 3: scratch dir (tmp and communication)
	tmpDir := filepath.Join(container.scratchDir, "tmp")
	if err := os.Mkdir(tmpDir, 0777); err != nil && !os.IsExist(err) {
		return err
	}

	sbScratchDir := filepath.Join(container.containerRootDir, "host")
	if err := syscall.Mount(container.scratchDir, sbScratchDir, "", common.BIND, ""); err != nil {
		return fmt.Errorf("failed to bind scratch dir: %v", err.Error())
	}

	// TODO: cheaper to handle with symlink in lambda image?
	sbTmpDir := filepath.Join(container.containerRootDir, "tmp")
	if err := syscall.Mount(tmpDir, sbTmpDir, "", common.BIND, ""); err != nil {
		return fmt.Errorf("failed to bind tmp dir: %v", err.Error())
	}

	return nil
}

func (container *SOCKContainer) Pause() (err error) {
	if err := container.cg.Pause(); err != nil {
		return err
	}

	if common.Conf.Features.Downsize_paused_mem {
		// drop mem limit to what is used when we're paused, because
		// we know the Sandbox cannot allocate more when it's not
		// schedulable.  Then release saved memory back to the pool.
		oldLimit := container.cg.getMemLimitMB()
		newLimit := container.cg.getMemUsageMB() + 1
		if newLimit < oldLimit {
			container.cg.setMemLimitMB(newLimit)
			container.pool.mem.adjustAvailableMB(oldLimit - newLimit)
		}
	}
	return nil
}

func (container *SOCKContainer) Unpause() (err error) {
	if common.Conf.Features.Downsize_paused_mem {
		// block until we have enough mem to upsize limit to the
		// normal size before unpausing
		oldLimit := container.cg.getMemLimitMB()
		newLimit := common.Conf.Limits.Mem_mb
		container.pool.mem.adjustAvailableMB(oldLimit - newLimit)
		container.cg.setMemLimitMB(newLimit)
	}

	return container.cg.Unpause()
}

func (container *SOCKContainer) Destroy() {
	if err := container.cg.Pause(); err != nil {
		panic(err)
	}

	container.decCgRefCount()
}

// when the count goes to zero, it means (a) this container and (b)
// all it's descendants are destroyed. Thus, it's safe to release it's
// cgroups, and return the memory allocation to the memPool
func (container *SOCKContainer) decCgRefCount() {
	container.cgRefCount -= 1
	container.printf("CG ref count decremented to %d", container.cgRefCount)
	if container.cgRefCount < 0 {
		panic("cgRefCount should not be able to go negative")
	}

	// release all resources when we have no more dependents...
	if container.cgRefCount == 0 {
		t := common.T0("Destroy()/kill-procs")
		if container.cg != nil {
			pids := container.cg.KillAllProcs()
			container.printf("killed PIDs %v in CG\n", pids)
		}
		t.T1()

		container.printf("unmount and remove dirs\n")
		t = common.T0("Destroy()/detach-root")
		if err := syscall.Unmount(container.containerRootDir, syscall.MNT_DETACH); err != nil {
			container.printf("unmount root dir %s failed :: %v\n", container.containerRootDir, err)
		}
		t.T1()

		t = common.T0("Destroy()/remove-root")
		if err := os.RemoveAll(container.containerRootDir); err != nil {
			container.printf("remove root dir %s failed :: %v\n", container.containerRootDir, err)
		}
		t.T1()

		container.cg.Release()
		container.pool.mem.adjustAvailableMB(container.cg.getMemLimitMB())

		if container.parent != nil {
			container.parent.childExit(container)
		}
	}
}

func (container *SOCKContainer) childExit(child Sandbox) {
	delete(container.children, child.ID())
	container.decCgRefCount()
}

// fork a new process from the Zygote in c, relocate it to be the server in dst
func (container *SOCKContainer) fork(dst Sandbox) (err error) {
	spareMB := container.cg.getMemLimitMB() - container.cg.getMemUsageMB()
	if spareMB < 3 {
		return fmt.Errorf("only %vMB of spare memory in parent, rejecting fork request (need at least 3MB)", spareMB)
	}

	dstSock := dst.(*safeSandbox).Sandbox.(*SOCKContainer)

	origPids, err := container.cg.GetPIDs()
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
	err = container.forkRequest(fmt.Sprintf("%s/ol.sock", container.scratchDir), root, memCG)
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
		currPids, err := container.cg.GetPIDs()
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
				container.printf("move PID %v from CG %v to CG %v\n", pid, container.cg.Name, dstSock.cg.Name)
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

	container.children[dst.ID()] = dst
	container.cgRefCount += 1
	return nil
}

func (container *SOCKContainer) Status(key SandboxStatus) (string, error) {
	switch key {
	case StatusMemFailures:
		return strconv.FormatBool(container.cg.ReadInt("memory", "memory.failcnt") > 0), nil
	default:
		return "", STATUS_UNSUPPORTED
	}
}

func (container *SOCKContainer) Meta() *SandboxMeta {
	return container.meta
}

func (container *SOCKContainer) GetRuntimeLog() string {
    data, err := ioutil.ReadFile(filepath.Join(container.scratchDir, "ol-runtime.log"))

    if err == nil {
        return string(data)
    } else {
        return ""
    }
}

func (container *SOCKContainer) DebugString() string {
	var s string = fmt.Sprintf("SOCK %s\n", container.ID())

	s += fmt.Sprintf("ROOT DIR: %s\n", container.containerRootDir)

	s += fmt.Sprintf("HOST DIR: %s\n", container.scratchDir)

	if pids, err := container.cg.GetPIDs(); err == nil {
		s += fmt.Sprintf("CGROUP PIDS: %s\n", strings.Join(pids, ", "))
	} else {
		s += fmt.Sprintf("CGROUP PIDS: unknown (%s)\n", err)
	}

	s += fmt.Sprintf("CGROUPS: %s\n", container.cg.Path("<RESOURCE>", ""))

	if state, err := ioutil.ReadFile(container.cg.Path("freezer", "freezer.state")); err == nil {
		s += fmt.Sprintf("FREEZE STATE: %s", state)
	} else {
		s += fmt.Sprintf("FREEZE STATE: unknown (%s)\n", err)
	}

	s += fmt.Sprintf("MEMORY USED: %d of %d MB\n",
		container.cg.getMemUsageMB(), container.cg.getMemLimitMB())

	s += fmt.Sprintf("MEMORY FAILURES: %d\n",
		container.cg.ReadInt("memory", "memory.failcnt"))

	return s
}

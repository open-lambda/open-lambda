package sandbox

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/ol/common"
	"github.com/open-lambda/open-lambda/ol/worker/sandbox/cgroups"
)

type SOCKContainer struct {
	pool             *SOCKPool
	id               string
	meta             *SandboxMeta
	containerRootDir string
	codeDir          string
	scratchDir       string
	cg               cgroups.Cgroup
	rtType           common.RuntimeType
	client           *http.Client

	Node     int
	IsZygote bool
	// 1 for self, plus 1 for each child (we can't release memory
	// until all descendants are dead, because they share the
	// pages of this Container, but this is the only container
	// charged)
	cgRefCount int32

	parent   Sandbox
	children map[string]Sandbox

	containerProxy *os.Process
}

// add ID to each log message so we know which logs correspond to
// which containers
func (container *SOCKContainer) printf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s [SOCK %s]", strings.TrimRight(msg, "\n"), container.id)
}

// ID returns the unique identifier of this container
func (container *SOCKContainer) ID() string {
	return container.id
}

func (container *SOCKContainer) GetRuntimeType() common.RuntimeType {
	return container.rtType
}

func (container *SOCKContainer) freshProc() (err error) {
	// get FD to cgroup
	cgFiles := make([]*os.File, 1)
	path := container.cg.CgroupProcsPath()
	fd, err := syscall.Open(path, syscall.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	cgFiles[0] = os.NewFile(uintptr(fd), path)
	defer cgFiles[0].Close()

	var cmd *exec.Cmd

	cgPath := "/default-ol-sandboxes/" + container.cg.Name()
	// todo: make this a Cgroup method.
	if container.rtType == common.RT_PYTHON {
		cmd = exec.Command(
			"sudo", "cgexec", "-g", "memory,cpu:"+cgPath,
			"chroot", container.containerRootDir, "python3", "-u",
			"/runtimes/python/server.py", "/host/bootstrap.py", strconv.Itoa(1),
			strconv.FormatBool(common.Conf.Features.Enable_seccomp),
		)
	} else if container.rtType == common.RT_NATIVE {
		if container.containerProxy == nil {
			err := container.launchContainerProxy()

			if err != nil {
				return err
			}
		}

		cmd = exec.Command(
			"chroot", container.containerRootDir,
			"env", "RUST_BACKTRACE=full", "/runtimes/native/server", strconv.Itoa(1),
			strconv.FormatBool(common.Conf.Features.Enable_seccomp),
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

	// Runtimes will fork off a new container,
	// so this won't block long
	return cmd.Wait()
}

func (container *SOCKContainer) launchContainerProxy() (err error) {
	args := []string{}
	args = append(args, "ol-container-proxy")
	args = append(args, container.scratchDir)

	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}

	proc, err := os.StartProcess("./ol-container-proxy", args, &procAttr)

	if err != nil {
		return fmt.Errorf("Failed to start database proxy")
	}

	died := make(chan error)
	go func() {
		_, err := proc.Wait()
		died <- err
	}()

	if err != nil {
		return fmt.Errorf("Failed to start database proxy")
	}

	var pingErr error

	//TODO make more efficient
	for i := 0; i < 300; i++ {
		// check if it has died
		select {
		case err := <-died:
			if err != nil {
				return err
			}
			return fmt.Errorf("container proxy does not seem to be running")
		default:
		}

		path := container.scratchDir + "/proxy.pid"
		data, err := os.ReadFile(path)

		if err != nil {
			pingErr = err
			time.Sleep(1 * time.Millisecond)
			continue
		}

		pid, err := strconv.Atoi(string(data))
		if err != nil {
			pingErr = err
			time.Sleep(1 * time.Millisecond)
			continue
		}

		if pid != proc.Pid {
			return fmt.Errorf("Database proxy pid does not match")
		}

		pingErr = nil
		container.containerProxy = proc
		break
	}

	return pingErr
}

func (container *SOCKContainer) populateRoot() (err error) {
	// FILE SYSTEM STEP 1: mount base
	baseDir := common.Conf.SOCK_base_path
	if err := syscall.Mount(baseDir, container.containerRootDir, "", common.BIND, ""); err != nil {
		return fmt.Errorf("failed to bind root dir: %s -> %s :: %v", baseDir, container.containerRootDir, err)
	}

	if err := syscall.Mount("none", container.containerRootDir, "", common.BIND_RO, ""); err != nil {
		return fmt.Errorf("failed to bind root dir RO: %s :: %v", container.containerRootDir, err)
	}

	if err := syscall.Mount("none", container.containerRootDir, "", common.PRIVATE, ""); err != nil {
		return fmt.Errorf("failed to make root dir private :: %v", err)
	}

	// todo: now the packages' dir are read-only, is neccessary to remount the packages dir using overlayfs?
	// todo: also, is it necessary to create a illusion like common site-packages dir?
	// create a dir used to hidden the content in packages dir
	//tmpEmptyDir, err := os.MkdirTemp("", "empty")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//if err := syscall.Mount(tmpEmptyDir, filepath.Join(container.containerRootDir, "packages"), "", common.BIND, ""); err != nil {
	//	return fmt.Errorf("failed to bind empty dir: %v", err)
	//}
	//
	//for _, pkg := range container.meta.Installs {
	//	srcDirStr := filepath.Join(common.Conf.SOCK_base_path, "packages", pkg, "files")
	//	targetDirStr := filepath.Join(container.containerRootDir, "packages", pkg, "files")
	//	err := os.MkdirAll(targetDirStr, 0777)
	//	if err != nil {
	//		return err
	//	}
	//
	//	if err := syscall.Mount(srcDirStr, targetDirStr, "", common.BIND, ""); err != nil {
	//		return fmt.Errorf("failed to bind package dir: %s -> %s :: %v", srcDirStr, targetDirStr, err)
	//	}
	//	if err := syscall.Mount("none", targetDirStr, "", common.BIND_RO, ""); err != nil {
	//		return fmt.Errorf("failed to bind package dir RO: %s :: %v", targetDirStr, err)
	//	}
	//	if err := syscall.Mount("none", targetDirStr, "", common.PRIVATE, ""); err != nil {
	//		return fmt.Errorf("failed to make package dir private :: %v", err)
	//	}
	//}

	// FILE SYSTEM STEP 2: code dir
	if container.codeDir != "" {
		sbCodeDir := filepath.Join(container.containerRootDir, "handler")

		if err := syscall.Mount(container.codeDir, sbCodeDir, "", common.BIND, ""); err != nil {
			return fmt.Errorf("Failed to bind code dir: %s -> %s :: %v", container.codeDir, sbCodeDir, err.Error())
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

// Pause stops/freezes the container
func (container *SOCKContainer) Pause() (err error) {
	if err := container.cg.Pause(); err != nil {
		return err
	}

	if common.Conf.Features.Downsize_paused_mem {
		// drop mem limit to what is used when we're paused, because
		// we know the Sandbox cannot allocate more when it's not
		// schedulable.  Then release saved memory back to the pool.
		oldLimit := container.cg.GetMemLimitMB()
		newLimit := container.cg.GetMemUsageMB() + 5 // 1 is not enough if we comment the setFreezeState
		if newLimit < oldLimit {
			container.cg.SetMemLimitMB(newLimit)
			container.pool.mem.adjustAvailableMB(oldLimit - newLimit)
		}
	}

	// save a little memory
	container.client.CloseIdleConnections()

	return nil
}

// Unpause resumes/unfreezes the container
func (container *SOCKContainer) Unpause() (err error) {
	if common.Conf.Features.Downsize_paused_mem {
		// block until we have enough mem to upsize limit to the
		// normal size before unpausing
		oldLimit := container.cg.GetMemLimitMB()
		newLimit := common.Conf.Limits.Mem_mb
		container.pool.mem.adjustAvailableMB(oldLimit - newLimit)
		container.cg.SetMemLimitMB(newLimit)
	}

	return container.cg.Unpause()
}

// Destroy shuts down the container
func (container *SOCKContainer) Destroy(reason string) {
	if err := container.cg.Pause(); err != nil {
		panic(err)
	}

	container.decCgRefCount()
}

func (container *SOCKContainer) DestroyIfPaused(reason string) {
	// we're allowed to implement this by unconditionally destroying
	container.Destroy(reason)
}

// when the count goes to zero, it means (a) this container and (b)
// all it's descendants are destroyed. Thus, it's safe to release it's
// cgroups, and return the memory allocation to the memPool
func (container *SOCKContainer) decCgRefCount() {
	newCount := atomic.AddInt32(&container.cgRefCount, -1)

	container.printf("CG ref count decremented to %d", newCount)
	if newCount < 0 {
		panic("cgRefCount should not be able to go negative")
	}

	// release all resources when we have no more dependents...
	if newCount == 0 {
		// Stop proxy before unmounting (because it might write to a logfile)
		if container.containerProxy != nil {
			container.containerProxy.Kill()
			container.containerProxy.Wait()
		}

		t := common.T0("Destroy()/cleanup-cgroup")
		if container.cg != nil {
			container.cg.KillAllProcs()
			container.printf("killed PIDs in CG\n")
			container.cg.Release()
			container.pool.mem.adjustAvailableMB(container.cg.GetMemLimitMB())
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

		if container.parent != nil {
			container.parent.childExit(container)
		}
	}
}

func (container *SOCKContainer) childExit(child Sandbox) {
	delete(container.children, child.ID())
	container.decCgRefCount()
}

// fork a new process from the Zygote in container, relocate it to be the server in dst
func (container *SOCKContainer) fork(dst Sandbox) (err error) {
	forkRec := make(map[string]int64)
	forkRec["get_mem"] = time.Now().UnixNano() / 1e6
	spareMB := container.cg.GetMemLimitMB() - container.cg.GetMemUsageMB()
	if spareMB < 3 {
		return fmt.Errorf("only %vMB of spare memory in parent, rejecting fork request (need at least 3MB)", spareMB)
	}
	forkRec["open_file"] = time.Now().UnixNano() / 1e6
	// increment reference count before we start any processes
	container.children[dst.ID()] = dst
	newCount := atomic.AddInt32(&container.cgRefCount, 1)

	if newCount == 0 {
		panic("cgRefCount was already 0")
	}

	dstSock := dst.(*SafeSandbox).Sandbox.(*SOCKContainer)

	root, err := os.Open(dstSock.containerRootDir)
	if err != nil {
		return err
	}
	defer root.Close()

	cg := dstSock.cg
	cgProcs, err := os.OpenFile(cg.CgroupProcsPath(), os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer cgProcs.Close()

	forkRec["fork_req"] = time.Now().UnixNano() / 1e6
	t := common.T0("forkRequest")
	err = container.forkRequest(fmt.Sprintf("%s/ol.sock", container.scratchDir), root, cgProcs)
	if err != nil {
		return err
	}
	t.T1()
	forkRec["fork_done"] = time.Now().UnixNano() / 1e6
	forkLock.Lock()
	ForkRecords = append(ForkRecords, forkRec)
	forkLock.Unlock()
	return nil
}

var forkLock = sync.Mutex{}
var ForkRecords = make([]map[string]int64, 0)

func (container *SOCKContainer) Meta() *SandboxMeta {
	return container.meta
}

func (container *SOCKContainer) Client() *http.Client {
	return container.client
}

// GetRuntimeLog returns the log of the runtime
func (container *SOCKContainer) GetRuntimeLog() string {
	data, err := ioutil.ReadFile(filepath.Join(container.scratchDir, "ol-runtime.log"))

	if err == nil {
		return string(data)
	}

	return ""
}

// GetProxyLog returns the log of the http proxy
func (container *SOCKContainer) GetProxyLog() string {
	data, err := ioutil.ReadFile(filepath.Join(container.scratchDir, "proxy.log"))

	if err == nil {
		return string(data)
	}

	return ""
}

func (container *SOCKContainer) DebugString() string {
	var s = fmt.Sprintf("SOCK %s\n", container.ID())
	s += fmt.Sprintf("ROOT DIR: %s\n", container.containerRootDir)
	s += fmt.Sprintf("HOST DIR: %s\n", container.scratchDir)
	s += container.cg.DebugString()
	return s
}

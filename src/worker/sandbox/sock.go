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
	"sync/atomic"
	"syscall"
	"time"
	"os/user"

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

	// 1 for self, plus 1 for each child (we can't release memory
	// until all descendants are dead, because they share the
	// pages of this Container, but this is the only container
	// charged)
	cgRefCount int32

	parent   Sandbox
	children map[string]Sandbox

	containerProxy *os.Process
}

// getUserSpec returns the user specification for chroot commands
func (container *SOCKContainer) getUserSpec() (string, error) {
	// Look up testuser
	testUser, err := user.Lookup("testuser")
	if err != nil {
		return "", fmt.Errorf("failed to lookup testuser: %v", err)
	}
	return fmt.Sprintf("%s:%s", testUser.Uid, testUser.Gid), nil
}

// ensureFilePermissions sets proper ownership for files that testuser needs access to
func (container *SOCKContainer) ensureFilePermissions() error {
	// Look up testuser
	testUser, err := user.Lookup("testuser")
	if err != nil {
		return fmt.Errorf("failed to lookup testuser: %v", err)
	}

	uid, err := strconv.Atoi(testUser.Uid)
	if err != nil {
		return fmt.Errorf("failed to parse testuser UID: %v", err)
	}

	gid, err := strconv.Atoi(testUser.Gid)
	if err != nil {
		return fmt.Errorf("failed to parse testuser GID: %v", err)
	}

	// Ensure testuser owns the scratch directory and its contents
	if err := os.Chown(container.scratchDir, uid, gid); err != nil {
		return fmt.Errorf("failed to chown scratch dir: %v", err)
	}

	// Create tmp directory with proper permissions
	tmpDir := filepath.Join(container.scratchDir, "tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create tmp dir: %v", err)
	}

	if err := os.Chown(tmpDir, uid, gid); err != nil {
		return fmt.Errorf("failed to chown tmp dir: %v", err)
	}

	// Recursively ensure all existing files in scratch directory are owned by testuser
	filepath.Walk(container.scratchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}
		os.Chown(path, uid, gid)
		// Also ensure files are readable by owner
		if !info.IsDir() {
			os.Chmod(path, 0644)
		} else {
			os.Chmod(path, 0755)
		}
		return nil
	})

	return nil
}

// ensureBootstrapPermissions specifically fixes bootstrap.py permissions if it exists
func (container *SOCKContainer) ensureBootstrapPermissions() error {
	bootstrapPath := filepath.Join(container.scratchDir, "bootstrap.py")

	// Check if bootstrap.py exists
	if _, err := os.Stat(bootstrapPath); err == nil {
		// File exists, fix its permissions
		testUser, err := user.Lookup("testuser")
		if err != nil {
			return fmt.Errorf("failed to lookup testuser: %v", err)
		}

		uid, err := strconv.Atoi(testUser.Uid)
		if err != nil {
			return fmt.Errorf("failed to parse testuser UID: %v", err)
		}

		gid, err := strconv.Atoi(testUser.Gid)
		if err != nil {
			return fmt.Errorf("failed to parse testuser GID: %v", err)
		}

		// Fix ownership and permissions
		if err := os.Chown(bootstrapPath, uid, gid); err != nil {
			return fmt.Errorf("failed to chown bootstrap.py: %v", err)
		}

		if err := os.Chmod(bootstrapPath, 0644); err != nil {
			return fmt.Errorf("failed to chmod bootstrap.py: %v", err)
		}

		container.printf("Fixed bootstrap.py permissions: owner=%d:%d, mode=0644", uid, gid)
	}

	return nil
}

// ensureContainerRootPermissions sets permissions on container root before mounting
func (container *SOCKContainer) ensureContainerRootPermissions() error {
	// Set permissions on container root directory before it gets mounted as read-only
	if err := os.Chmod(container.containerRootDir, 0755); err != nil {
		return fmt.Errorf("failed to chmod container root dir: %v", err)
	}

	return nil
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
	// Ensure bootstrap.py has correct permissions if it exists
	if err := container.ensureBootstrapPermissions(); err != nil {
		container.printf("Warning: failed to fix bootstrap permissions: %v", err)
	}

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

	if container.rtType == common.RT_PYTHON {
		userSpec, err := container.getUserSpec()
		if err != nil {
			return fmt.Errorf("failed to get user spec: %v", err)
		}
		cmd = exec.Command(
			"chroot", "--userspec="+userSpec, container.containerRootDir, "python3", "-u",
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

		userSpec, err := container.getUserSpec()
		if err != nil {
			return fmt.Errorf("failed to get user spec: %v", err)
		}

		cmd = exec.Command(
			"chroot", "--userspec="+userSpec, container.containerRootDir,
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
	err = cmd.Wait()

	// Debug: Check what's in the scratch directory after process completes
	container.printf("Process completed, checking scratch directory contents:")
	if files, readErr := os.ReadDir(container.scratchDir); readErr == nil {
		for _, file := range files {
			container.printf("  %s", file.Name())
		}
	} else {
		container.printf("  Error reading scratch dir: %v", readErr)
	}

	// Specifically check for ol.sock
	sockPath := filepath.Join(container.scratchDir, "ol.sock")
	if _, statErr := os.Stat(sockPath); statErr == nil {
		container.printf("ol.sock exists at %s", sockPath)
	} else {
		container.printf("ol.sock NOT found at %s: %v", sockPath, statErr)
	}

	return err
}

func (container *SOCKContainer) launchContainerProxy() (err error) {
	args := []string{}
	args = append(args, "ol-container-proxy")
	args = append(args, container.scratchDir)

	var procAttr os.ProcAttr
	procAttr.Files = []*os.File{os.Stdin, os.Stdout, os.Stderr}

	binPath, err := exec.LookPath("ol-container-proxy")
	if err != nil {
		return fmt.Errorf("Failed to find container proxy binary: %s", err)
	}

	proc, err := os.StartProcess(binPath, args, &procAttr)

	if err != nil {
		return fmt.Errorf("Failed to start container proxy: %s", err)
	}

	died := make(chan error)
	go func() {
		_, err := proc.Wait()
		died <- err
	}()

	if err != nil {
		return fmt.Errorf("Failed to start container proxy: %s", err)
	}

	var pingErr error

	// TODO make this more efficient
	for i := 0; i < 300; i++ {
		// check if it has died
		select {
		case err := <-died:
			if err != nil {
				return err
			}
			return fmt.Errorf("Container proxy does not seem to be running")
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
			return fmt.Errorf("Container proxy pid does not match")
		}

		pingErr = nil
		container.containerProxy = proc
		break
	}

	return pingErr
}

func (container *SOCKContainer) populateRoot() (err error) {
	// Ensure proper permissions before mounting
	if err := container.ensureContainerRootPermissions(); err != nil {
		return fmt.Errorf("failed to set container root permissions: %v", err)
	}

	if err := container.ensureFilePermissions(); err != nil {
		return fmt.Errorf("failed to set file permissions: %v", err)
	}

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
	if err := os.MkdirAll(tmpDir, 0755); err != nil && !os.IsExist(err) {
		return err
	}

	// Ensure testuser owns the tmp directory
	testUser, err := user.Lookup("testuser")
	if err == nil {
		if uid, uidErr := strconv.Atoi(testUser.Uid); uidErr == nil {
			if gid, gidErr := strconv.Atoi(testUser.Gid); gidErr == nil {
				os.Chown(tmpDir, uid, gid)
			}
		}
	}

	sbScratchDir := filepath.Join(container.containerRootDir, "host")
	if err := os.MkdirAll(sbScratchDir, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create host mount point: %v", err)
	}
	if err := syscall.Mount(container.scratchDir, sbScratchDir, "", common.BIND, ""); err != nil {
		return fmt.Errorf("failed to bind scratch dir: %v", err.Error())
	}

	// DO NOT make the /host mount read-only - we need to create sockets there
	// This is different from the code and base mounts which are read-only for security

	// Ensure the mounted /host directory has proper permissions for testuser
	if testUser, err := user.Lookup("testuser"); err == nil {
		if uid, uidErr := strconv.Atoi(testUser.Uid); uidErr == nil {
			if gid, gidErr := strconv.Atoi(testUser.Gid); gidErr == nil {
				// Note: We can't chmod the mounted directory after it's mounted as read-only
				// But we can ensure the source directory has proper permissions
				os.Chown(container.scratchDir, uid, gid)
				os.Chmod(container.scratchDir, 0755)
				container.printf("Set scratch dir permissions for testuser: %d:%d", uid, gid)
			}
		}
	}

	// TODO: cheaper to handle with symlink in lambda image?
	sbTmpDir := filepath.Join(container.containerRootDir, "tmp")
	if err := syscall.Mount(tmpDir, sbTmpDir, "", common.BIND, ""); err != nil {
		return fmt.Errorf("failed to bind tmp dir: %v", err.Error())
	}

	container.printf("Mount setup complete - scratch=%s mounted to /host", container.scratchDir)

	// Debug: check what's in the scratch directory before socket creation
	if files, err := os.ReadDir(container.scratchDir); err == nil {
		container.printf("Scratch directory contents before socket creation:")
		for _, file := range files {
			container.printf("  %s", file.Name())
		}
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
		newLimit := container.cg.GetMemUsageMB() + 1
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
func (container *SOCKContainer) Destroy(_ string) {
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
	spareMB := container.cg.GetMemLimitMB() - container.cg.GetMemUsageMB()
	if spareMB < 3 {
		return fmt.Errorf("only %vMB of spare memory in parent, rejecting fork request (need at least 3MB)", spareMB)
	}

	// increment reference count before we start any processes
	container.children[dst.ID()] = dst
	newCount := atomic.AddInt32(&container.cgRefCount, 1)

	if newCount == 0 {
		panic("cgRefCount was already 0")
	}

	dstSock := dst.(*safeSandbox).Sandbox.(*SOCKContainer)
	// Set up the destination container's filesystem BEFORE forking
	container.printf("Setting up destination container filesystem for fork")
	if err := dstSock.populateRoot(); err != nil {
		return fmt.Errorf("failed to setup destination container root: %v", err)
	}

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
	cgProcs, err := os.OpenFile(cg.CgroupProcsPath(), os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer cgProcs.Close()

	t := common.T0("forkRequest")
	err = container.forkRequest(fmt.Sprintf("%s/ol.sock", container.scratchDir), root, cgProcs)
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
				container.printf("move PID %v from CG %v to CG %v\n", pid, container.cg.Name(), dstSock.cg.Name())
				if err = dstSock.cg.AddPid(pid); err != nil {
					return err
				}
				moved++
			}
		}

		if moved == 0 {
			break
		}
	}
	t.T1()

	// Wait a bit and check if the socket was created in the destination
	time.Sleep(100 * time.Millisecond)
	expectedSockPath := filepath.Join(dstSock.scratchDir, "ol.sock")
	if _, err := os.Stat(expectedSockPath); err != nil {
		container.printf("Warning: socket not found at expected location %s: %v", expectedSockPath, err)
		// List contents of destination scratch directory
		if files, readErr := os.ReadDir(dstSock.scratchDir); readErr == nil {
			container.printf("Destination scratch directory contents:")
			for _, file := range files {
				container.printf("  %s", file.Name())
			}
		}
	} else {
		container.printf("Fork successful: socket created at %s", expectedSockPath)
	}

	return nil
}

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

package sandbox

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/ol/config"
)

// the first program is executed on the host, which sets up the
// container, running the second program inside the container
const SOCK_HOST_INIT = "/usr/local/bin/sock-init"
const SOCK_GUEST_INIT = "/ol-init"

var BIND uintptr = uintptr(syscall.MS_BIND)
var BIND_RO uintptr = uintptr(syscall.MS_BIND | syscall.MS_RDONLY | syscall.MS_REMOUNT)
var PRIVATE uintptr = uintptr(syscall.MS_PRIVATE)
var SHARED uintptr = uintptr(syscall.MS_SHARED)

var nextId int64 = 0

// SOCKPool is a ContainerFactory that creats docker containeres.
type SOCKPool struct {
	name         string
	cgPool       *CgroupPool
	rootDir      string
	unshareFlags string
	initArgs     []string

	sync.Mutex
	sandboxes []Sandbox
}

// NewSOCKPool creates a SOCKPool.
func NewSOCKPool(rootDir string, isImportCache bool) (cf *SOCKPool, err error) {
	var unshareFlags string
	var initArgs []string
	var name string

	if isImportCache {
		// we cannot move processes forked in the import cache
		// across PID namespaces
		unshareFlags = "-iu"
		initArgs = []string{"--cache"}
		name = "sock-cache"
	} else {
		unshareFlags = "-ipu"
		initArgs = []string{}
		name = "sock-handlers"
	}

	cgPool := NewCgroupPool(name)

	if err := os.MkdirAll(rootDir, 0777); err != nil {
		return nil, fmt.Errorf("failed to make root container dir :: %v", err)
	}

	if err := syscall.Mount(rootDir, rootDir, "", BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root container dir: %v", err)
	}

	if err := syscall.Mount("none", rootDir, "", PRIVATE, ""); err != nil {
		return nil, fmt.Errorf("failed to make root container dir private :: %v", err)
	}

	pool := &SOCKPool{
		name:         name,
		cgPool:       cgPool,
		rootDir:      rootDir,
		initArgs:     initArgs,
		unshareFlags: unshareFlags,
	}

	return pool, nil
}

func (pool *SOCKPool) Create(codeDir, scratchPrefix string, imports []string) (Sandbox, error) {
	return pool.CreateFromParent(codeDir, scratchPrefix, imports, nil)
}

func (pool *SOCKPool) CreateFromParent(codeDir, scratchPrefix string, imports []string, parent Sandbox) (sb Sandbox, err error) {
	if config.Conf.Timing {
		defer func(start time.Time) {
			log.Printf("create sock took %v\n", time.Since(start))
		}(time.Now())
	}

	log.Printf("%v.CreateFromParent(%v, %v, %v, %v)", pool.name, codeDir, scratchPrefix, imports, parent)

	id := fmt.Sprintf("%d", atomic.AddInt64(&nextId, 1))
	containerRootDir := filepath.Join(pool.rootDir, id)
	scratchDir := filepath.Join(scratchPrefix, id)

	startCmd := append([]string{SOCK_GUEST_INIT}, pool.initArgs...)

	var c *SOCKContainer = &SOCKContainer{
		id:               id,
		containerRootDir: containerRootDir,
		codeDir:          codeDir,
		scratchDir:       scratchDir,
		unshareFlags:     pool.unshareFlags,
	}

	defer func() {
		if err != nil {
			c.Destroy()
		}
	}()

	// general setup (all SOCK sandboxes do this)
	if err := c.start(startCmd, pool.cgPool); err != nil {
		return nil, fmt.Errorf("failed to start: %v", err)
	}

	// specific setup (may actually start running user-supplied code)
	if parent != nil {
		if err := parent.fork(c, imports, true); err != nil {
			return nil, err
		}
	} else {
		if err := c.runServer(); err != nil {
			return nil, err
		}
	}

	if err := waitForServerPipeReady(c.HostDir()); err != nil {
		return nil, err
	}

	// wrap to make thread-safe and handle container death
	safeSB := &safeSandbox{Sandbox: c}

	// TODO: have some way to clean up this structure as sandboxes are released
	pool.Mutex.Lock()
	pool.sandboxes = append(pool.sandboxes, safeSB)
	pool.Mutex.Unlock()

	return safeSB, nil
}

func (pool *SOCKPool) Cleanup() {
	pool.Mutex.Lock()
	for _, sandbox := range pool.sandboxes {
		sandbox.Destroy()
	}
	pool.Mutex.Unlock()

	pool.cgPool.Destroy()
	syscall.Unmount(pool.rootDir, syscall.MNT_DETACH)
	os.RemoveAll(pool.rootDir)
}

func (pool *SOCKPool) DebugString() string {
	pool.Mutex.Lock()
	defer pool.Mutex.Unlock()

	var sb strings.Builder

	for _, sandbox := range pool.sandboxes {
		sb.WriteString(fmt.Sprintf("%s--------\n", sandbox.DebugString()))
	}

	return sb.String()
}

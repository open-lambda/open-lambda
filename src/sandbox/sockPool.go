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
	name          string
	rootDir       string
	cgPool        *CgroupPool
	mem           *MemPool
	eventHandlers []SandboxEventFunc

	sync.Mutex
	sandboxes []Sandbox
}

// NewSOCKPool creates a SOCKPool.
func NewSOCKPool(name string, mem *MemPool) (cf *SOCKPool, err error) {
	cgPool, err := NewCgroupPool(name)
	if err != nil {
		return nil, err
	}

	rootDir := filepath.Join(config.Conf.Worker_dir, name)

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
		name:          name,
		mem:           mem,
		cgPool:        cgPool,
		rootDir:       rootDir,
		eventHandlers: []SandboxEventFunc{},
	}

	return pool, nil
}

func (pool *SOCKPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchPrefix string, imports []string) (sb Sandbox, err error) {
	if config.Conf.Timing {
		defer func(start time.Time) {
			log.Printf("create sock took %v\n", time.Since(start))
		}(time.Now())
	}

	log.Printf("<%v>.Create(%v, %v, %v, %v)", pool.name, codeDir, scratchPrefix, imports, parent)

	// block until we have enough to cover the cgroup mem limits
	pool.mem.adjustAvailableMB(-config.Conf.Sock_cgroups.Max_mem_mb)

	id := fmt.Sprintf("%d", atomic.AddInt64(&nextId, 1))
	containerRootDir := filepath.Join(pool.rootDir, id)
	scratchDir := filepath.Join(scratchPrefix, id)

	var c *SOCKContainer = &SOCKContainer{
		pool:             pool,
		id:               id,
		containerRootDir: containerRootDir,
		codeDir:          codeDir,
		scratchDir:       scratchDir,
		cg:               pool.cgPool.GetCg(),
	}

	defer func() {
		if err != nil {
			c.Destroy()
		}
	}()

	// root file system
	if err := c.populateRoot(); err != nil {
		return nil, fmt.Errorf("failed to create root FS: %v", err)
	}

	// write the Python code that the new process will run when it starts
	var pyCode []string
	if isLeaf {
		pyCode = []string{
			"sys.path.extend(['/packages', '/handler'])",
			"web_server('/host/ol.sock')",
		}
	} else {
		pyCode = []string{
			"sys.path.extend(['/packages'])",
			"fork_server('/host/ol.sock')",
		}
	}
	if err := c.writeBootstrapCode(pyCode); err != nil {
		return nil, err
	}

	// create new process in container (fresh, or forked from parent)
	if parent == nil {
		if err := c.freshProc(); err != nil {
			return nil, err
		}
	} else {
		if err := parent.fork(c); err != nil {
			return nil, err
		}
	}

	// wrap to make thread-safe and handle container death
	safeSB := newSafeSandbox(c, pool.eventHandlers)

	// TODO: have some way to clean up this structure as sandboxes are released
	pool.Mutex.Lock()
	pool.sandboxes = append(pool.sandboxes, safeSB)
	pool.Mutex.Unlock()

	return safeSB, nil
}

// handler(...) will be called everytime a sandbox-related event occurs,
// such as Create, Destroy, etc.
//
// the events are sent after the actions complete
//
// TODO: eventually make this part of SandboxPool API, and support in Docker?
func (pool *SOCKPool) AddListener(handler SandboxEventFunc) {
	pool.eventHandlers = append(pool.eventHandlers, handler)
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

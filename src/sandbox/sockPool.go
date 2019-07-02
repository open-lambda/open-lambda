package sandbox

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/stats"
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
	debugger
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

	pool.debugger = newDebugger(pool)

	return pool, nil
}

func (pool *SOCKPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, imports []string) (sb Sandbox, err error) {
	log.Printf("<%v>.Create(%v, %v, %v, %v)...", pool.name, codeDir, scratchDir, imports, parent)
	defer func() {
		log.Printf("...returns %v, %v", sb, err)
	}()

	t := stats.T0("Create()")
	defer t.T1()

	// block until we have enough to cover the cgroup mem limits
	t2 := t.T0("acquire-mem")
	pool.mem.adjustAvailableMB(-config.Conf.Sock_cgroups.Max_mem_mb)
	t2.T1()

	id := fmt.Sprintf("%d", atomic.AddInt64(&nextId, 1))
	containerRootDir := filepath.Join(pool.rootDir, id)

	t2 = t.T0("acquire-cgroup")
	cg := pool.cgPool.GetCg()
	t2.T1()

	var c *SOCKContainer = &SOCKContainer{
		pool:             pool,
		id:               id,
		containerRootDir: containerRootDir,
		codeDir:          codeDir,
		scratchDir:       scratchDir,
		cg:               cg,
	}

	defer func() {
		if err != nil {
			c.Destroy()
		}
	}()

	// root file system
	if isLeaf && c.codeDir == "" {
		return nil, fmt.Errorf("leaf sandboxes must have codeDir set")
	}

	t2 = t.T0("make-root-fs")
	if err := c.populateRoot(); err != nil {
		return nil, fmt.Errorf("failed to create root FS: %v", err)
	}
	t2.T1()

	// write the Python code that the new process will run when it starts
	var pyCode []string
	if isLeaf {
		pyCode = []string{
			"sys.path.extend(['/packages', '/handler'])",
			"web_server()",
		}
	} else {
		pyCode = []string{
			"sys.path.extend(['/packages'])",
			"fork_server()",
		}
	}
	if err := c.writeBootstrapCode(pyCode); err != nil {
		return nil, err
	}

	// create new process in container (fresh, or forked from parent)
	err = nil
	if parent != nil {
		t2 := t.T0("fork-proc")
		if err = parent.fork(c); err != nil {
			if err == DEAD_SANDBOX {
				log.Printf("parent SB %s died, create child with nil parent", parent.ID())
				parent = nil
			} else {
				return nil, err
			}
		}
		t2.T1()
	}

	if parent == nil {
		t2 := t.T0("fresh-proc")
		if err := c.freshProc(); err != nil {
			return nil, err
		}
		t2.T1()
	}

	// wrap to make thread-safe and handle container death
	return newSafeSandbox(c, pool.eventHandlers), nil
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
	// user is required to kill all containers before they call
	// this.  If they did, the memory pool should be full.
	log.Printf("SOCKPool.Cleanup: make sure all memory is free")
	pool.mem.adjustAvailableMB(-pool.mem.totalMB)
	log.Printf("SOCKPool.Cleanup: memory pool emptied")

	pool.cgPool.Destroy()
	syscall.Unmount(pool.rootDir, syscall.MNT_DETACH)
	os.RemoveAll(pool.rootDir)
}

func (pool *SOCKPool) DebugString() string {
	return pool.debugger.Dump()
}

func setSubtract(a []string, b []string) []string {
	rv := make([]string, len(a))
	for _, v1 := range a {
		inB := false
		for _, v2 := range b {
			if v2 == v1 {
				inB = true
			}
		}
		if !inB {
			rv = append(rv, v1)
		}
	}
	return rv
}

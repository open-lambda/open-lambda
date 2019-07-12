package sandbox

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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

func sbStr(sb Sandbox) string {
	if sb == nil {
		return "<nil>"
	}
	return fmt.Sprintf("<SB %s>", sb.ID())
}

func (pool *SOCKPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta) (sb Sandbox, err error) {
	id := fmt.Sprintf("%d", atomic.AddInt64(&nextId, 1))
	meta = fillMetaDefaults(meta)
	pool.printf("<%v>.Create(%v, %v, %v, %v, %v)=%s...", pool.name, sbStr(parent), isLeaf, codeDir, scratchDir, meta, id)
	defer func() {
		pool.printf("...returns %v, %v", sbStr(sb), err)
	}()

	t := stats.T0("Create()")
	defer t.T1()

	// block until we have enough to cover the cgroup mem limits
	t2 := t.T0("acquire-mem")
	pool.mem.adjustAvailableMB(-meta.MemLimitMB)
	t2.T1()

	t2 = t.T0("acquire-cgroup")
	cg := pool.cgPool.GetCg(meta.MemLimitMB)
	t2.T1()

	containerRootDir := filepath.Join(pool.rootDir, id)
	var cSock *SOCKContainer = &SOCKContainer{
		pool:             pool,
		id:               id,
		containerRootDir: containerRootDir,
		codeDir:          codeDir,
		scratchDir:       scratchDir,
		cg:               cg,
		children:         make([]Sandbox, 0),
		meta:             meta,
	}
	var c Sandbox = cSock

	defer func() {
		if err != nil {
			c.Destroy()
		}
	}()

	// root file system
	if isLeaf && cSock.codeDir == "" {
		return nil, fmt.Errorf("leaf sandboxes must have codeDir set")
	}

	t2 = t.T0("make-root-fs")
	if err := cSock.populateRoot(); err != nil {
		return nil, fmt.Errorf("failed to create root FS: %v", err)
	}
	t2.T1()

	// write the Python code that the new process will run when it starts
	var pyCode []string
	if isLeaf {
		pyCode = append(pyCode, "web_server()")
	} else {
		for _, mod := range meta.Imports {
			pool.printf("Pre-import module '%s'", mod)
			pyCode = append(pyCode, fmt.Sprintf("import %s", mod))
		}
		pyCode = append(pyCode, "fork_server()")
	}
	if err := cSock.writeBootstrapCode(pyCode); err != nil {
		return nil, err
	}

	// wrap to make thread-safe and handle container death.
	// after this line, two things happen:
	// 1. listeners (e.g., evictors) become aware of the Sandbox
	// 2. if this function fails, .Destroy() cleanup will be through the safeSandbox layer
	c = newSafeSandbox(c, pool.eventHandlers)

	// create new process in container (fresh, or forked from parent)
	if parent != nil {
		t2 := t.T0("fork-proc")
		if err := parent.fork(c); err != nil {
			if err != nil {
				pool.printf("parent.fork returned %v", err)
				return nil, FORK_FAILED
			}
		}
		t2.T1()
	} else {
		t2 := t.T0("fresh-proc")
		if err := cSock.freshProc(); err != nil {
			return nil, err
		}
		t2.T1()
	}

	return c, nil
}

func (pool *SOCKPool) printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s [SOCK POOL %s]", strings.TrimRight(msg, "\n"), pool.name)
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
	pool.printf("make sure all memory is free")
	pool.mem.adjustAvailableMB(-pool.mem.totalMB)
	pool.printf("memory pool emptied")

	pool.cgPool.Destroy()
	syscall.Unmount(pool.rootDir, syscall.MNT_DETACH)
	os.RemoveAll(pool.rootDir)
}

func (pool *SOCKPool) DebugString() string {
	return pool.debugger.Dump()
}

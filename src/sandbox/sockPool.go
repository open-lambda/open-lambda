package sandbox

import (
	"container/list"
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
	name    string
	cgPool  *CgroupPool
	rootDir string

	// a task listens on this, with requests to decrement memory
	// (which may block) or increment it
	memRequests chan *memReq

	// decrement requests read from memRequests that need to wait
	// for memory sit here until it's available
	memRequestsWaiting *list.List

	sync.Mutex
	sandboxes []Sandbox
}

type memReq struct {
	// how much we're requesting
	mb int

	// any response means the memory is allocated; the particular
	// number indicates the total remaining memory available in
	// the pool
	resp chan int
}

// NewSOCKPool creates a SOCKPool.
func NewSOCKPool(name string, pool_size_mb int) (cf *SOCKPool, err error) {
	cgPool := NewCgroupPool(name)
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
		name:               name,
		cgPool:             cgPool,
		rootDir:            rootDir,
		memRequests:        make(chan *memReq, 32),
		memRequestsWaiting: list.New(),
	}

	go pool.memTask(pool_size_mb)

	return pool, nil
}

// this task is responsible for tracking available memory in the
// system, adding to the count when memory is released, and blocking
// requesters until enough is free
func (pool *SOCKPool) memTask(pool_size_mb int) {
	available_mb := pool_size_mb

	for {
		req, ok := <-pool.memRequests
		if !ok {
			return
		}

		if req.mb >= 0 {
			available_mb += req.mb
			req.resp <- available_mb
		} else {
			pool.memRequestsWaiting.PushBack(req)
		}

		if e := pool.memRequestsWaiting.Front(); e != nil {
			req = e.Value.(*memReq)
			if available_mb+req.mb >= 0 {
				pool.memRequestsWaiting.Remove(e)
				available_mb += req.mb
				req.resp <- available_mb
			}
		}
	}
}

// this adjusts the available memory in the pool up/down, and returns
// the remaining available after the adjustment.
//
// Available memory is kept >=0, so a negative mb may block for some
// time.
//
// Sending a mb of 0 is a reasonable use case, especially for an
// evictor (it doesn't change anything, but provides a way to monitor
// available memory).
func (pool *SOCKPool) adjustAvailableMemMB(mb int) (available_mb int) {
	req := &memReq{
		mb:   mb,
		resp: make(chan int),
	}

	pool.memRequests <- req
	return <-req.resp
}

func (pool *SOCKPool) Create(parent Sandbox, isLeaf bool, codeDir, scratchPrefix string, imports []string) (sb Sandbox, err error) {
	if config.Conf.Timing {
		defer func(start time.Time) {
			log.Printf("create sock took %v\n", time.Since(start))
		}(time.Now())
	}

	log.Printf("<%v>.Create(%v, %v, %v, %v)", pool.name, codeDir, scratchPrefix, imports, parent)

	// block until we have enough to cover the cgroup mem limits
	pool.adjustAvailableMemMB(-config.Conf.Sock_cgroups.Max_mem_mb)

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
	close(pool.memRequests)
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

package sandbox

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/ol/config"
)

const OL_INIT = "/ol-init"

var BIND uintptr = uintptr(syscall.MS_BIND)
var BIND_RO uintptr = uintptr(syscall.MS_BIND | syscall.MS_RDONLY | syscall.MS_REMOUNT)
var PRIVATE uintptr = uintptr(syscall.MS_PRIVATE)
var SHARED uintptr = uintptr(syscall.MS_SHARED)

// SOCKPool is a ContainerFactory that creats docker containeres.
type SOCKPool struct {
	cgPool       *CgroupPool
	idxPtr       *int64
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
	var cgPool *CgroupPool

	if isImportCache {
		// we cannot move processes forked in the import cache
		// across PID namespaces
		unshareFlags = "-iu"
		initArgs = []string{"--cache"}
		cgPool = NewCgroupPool("sock-cache")
	} else {
		unshareFlags = "-ipu"
		initArgs = []string{}
		cgPool = NewCgroupPool("sock-handlers")
	}

	if err := os.MkdirAll(rootDir, 0777); err != nil {
		return nil, fmt.Errorf("failed to make root container dir :: %v", err)
	}

	if err := syscall.Mount(rootDir, rootDir, "", BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root container dir: %v", err)
	}

	if err := syscall.Mount("none", rootDir, "", PRIVATE, ""); err != nil {
		return nil, fmt.Errorf("failed to make root container dir private :: %v", err)
	}

	var sharedIdx int64 = -1
	idxPtr := &sharedIdx

	pool := &SOCKPool{
		cgPool:       cgPool,
		idxPtr:       idxPtr,
		rootDir:      rootDir,
		initArgs:     initArgs,
		unshareFlags: unshareFlags,
	}

	return pool, nil
}

func (pool *SOCKPool) Create(codeDir, workingDir string, imports []string) (Sandbox, error) {
	return pool.CreateFromParent(codeDir, workingDir, imports, nil)
}

func (pool *SOCKPool) CreateFromParent(codeDir, workingDir string, imports []string, parent Sandbox) (Sandbox, error) {
	if config.Conf.Timing {
		defer func(start time.Time) {
			log.Printf("create sock took %v\n", time.Since(start))
		}(time.Now())
	}

	id := fmt.Sprintf("%d", atomic.AddInt64(pool.idxPtr, 1))
	containerRootDir := filepath.Join(pool.rootDir, id)
	scratchDir := filepath.Join(workingDir, id)

	startCmd := append([]string{OL_INIT}, pool.initArgs...)

	c, err := NewSOCKContainer(id, containerRootDir, codeDir, scratchDir, pool.cgPool,
		pool.unshareFlags, startCmd, parent, imports)

	if err != nil {
		return nil, err
	}

	// TODO: have some way to clean up this structure as sandboxes are released
	pool.Mutex.Lock()
	pool.sandboxes = append(pool.sandboxes, c)
	pool.Mutex.Unlock()

	return c, nil
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

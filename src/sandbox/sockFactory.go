package sandbox

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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

// SOCKContainerFactory is a ContainerFactory that creats docker containeres.
type SOCKContainerFactory struct {
	cgPool       *CgroupPool
	idxPtr       *int64
	rootDir      string
	unshareFlags string
	initArgs     []string
}

// NewSOCKContainerFactory creates a SOCKContainerFactory.
func NewSOCKContainerFactory(rootDir string, isImportCache bool) (cf *SOCKContainerFactory, err error) {
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

	sf := &SOCKContainerFactory{
		cgPool:       cgPool,
		idxPtr:       idxPtr,
		rootDir:      rootDir,
		initArgs:     initArgs,
		unshareFlags: unshareFlags,
	}

	return sf, nil
}

// Create creates a docker container from the handler and container directory.
func (sf *SOCKContainerFactory) Create(codeDir, workingDir string, imports []string) (Sandbox, error) {
	return sf.CreateFromParent(codeDir, workingDir, imports, nil)
}

func (sf *SOCKContainerFactory) CreateFromParent(codeDir, workingDir string, imports []string, parent *SOCKContainer) (Sandbox, error) {
	if config.Conf.Timing {
		defer func(start time.Time) {
			log.Printf("create sock took %v\n", time.Since(start))
		}(time.Now())
	}

	id := fmt.Sprintf("%d", atomic.AddInt64(sf.idxPtr, 1))
	containerRootDir := filepath.Join(sf.rootDir, id)
	scratchDir := filepath.Join(workingDir, id)

	startCmd := append([]string{OL_INIT}, sf.initArgs...)
	return NewSOCKContainer(id, containerRootDir, codeDir, scratchDir, sf.cgPool,
		sf.unshareFlags, startCmd, parent, imports)
}

func (sf *SOCKContainerFactory) Cleanup() {
	sf.cgPool.Destroy()
	syscall.Unmount(sf.rootDir, syscall.MNT_DETACH)
	os.RemoveAll(sf.rootDir)
}

package sandbox

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
)

const OL_INIT = "/ol-init"

var BIND uintptr = uintptr(syscall.MS_BIND)
var BIND_RO uintptr = uintptr(syscall.MS_BIND | syscall.MS_RDONLY | syscall.MS_REMOUNT)
var PRIVATE uintptr = uintptr(syscall.MS_PRIVATE)
var SHARED uintptr = uintptr(syscall.MS_SHARED)

// SOCKContainerFactory is a ContainerFactory that creats docker containeres.
type SOCKContainerFactory struct {
	opts         *config.Config
	cgf          *CgroupFactory
	idxPtr       *int64
	rootDir      string
	baseDir      string
	unshareFlags string
	initArgs     []string
}

// NewSOCKContainerFactory creates a SOCKContainerFactory.
func NewSOCKContainerFactory(opts *config.Config, rootDir, prefix, unshareFlags string, initArgs []string) (*SOCKContainerFactory, error) {
	baseDir := opts.SOCK_base_path

	if err := os.MkdirAll(rootDir, 0777); err != nil {
		return nil, fmt.Errorf("failed to make root container dir :: %v", err)
	}

	if err := syscall.Mount(rootDir, rootDir, "", BIND, ""); err != nil {
		return nil, fmt.Errorf("failed to bind root container dir: %v", err)
	}

	if err := syscall.Mount("none", rootDir, "", PRIVATE, ""); err != nil {
		return nil, fmt.Errorf("failed to make root container dir private :: %v", err)
	}

	cgf, err := NewCgroupFactory(prefix, opts.Cg_pool_size)
	if err != nil {
		return nil, err
	}

	var sharedIdx int64 = -1
	idxPtr := &sharedIdx

	sf := &SOCKContainerFactory{
		opts:         opts,
		cgf:          cgf,
		idxPtr:       idxPtr,
		rootDir:      rootDir,
		baseDir:      baseDir,
		initArgs:     initArgs,
		unshareFlags: unshareFlags,
	}

	return sf, nil
}

// Create creates a docker container from the handler and container directory.
func (sf *SOCKContainerFactory) Create(codeDir, workingDir string) (Container, error) {
	if config.Timing {
		defer func(start time.Time) {
			log.Printf("create sock took %v\n", time.Since(start))
		}(time.Now())
	}

	id := fmt.Sprintf("%d", atomic.AddInt64(sf.idxPtr, 1))
	containerRootDir := filepath.Join(sf.rootDir, id)
	scratchDir := filepath.Join(workingDir, id)

	startCmd := append([]string{OL_INIT}, sf.initArgs...)
	return NewSOCKContainer(id, containerRootDir, sf.baseDir, codeDir, scratchDir, sf.cgf, sf.opts, sf.unshareFlags, startCmd), nil
}

func (sf *SOCKContainerFactory) Cleanup() {
	for _, cgroup := range CGroupList {
		cgroupPath := filepath.Join("/sys/fs/cgroup", cgroup, OLCGroupName)
		os.RemoveAll(cgroupPath)
	}

	syscall.Unmount(sf.rootDir, syscall.MNT_DETACH)
	os.RemoveAll(sf.rootDir)
}

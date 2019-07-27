package common

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
)

type StoreMode int

const (
	STORE_REGULAR StoreMode = iota
	STORE_PRIVATE
	STORE_MEMORY
)

var BIND uintptr = uintptr(syscall.MS_BIND)
var BIND_RO uintptr = uintptr(syscall.MS_BIND | syscall.MS_RDONLY | syscall.MS_REMOUNT)
var PRIVATE uintptr = uintptr(syscall.MS_PRIVATE)
var SHARED uintptr = uintptr(syscall.MS_SHARED)

var nextDirId int64 = 1000

type DirMaker struct {
	prefix string
	mode   StoreMode
}

func NewDirMaker(system string, mode StoreMode) (*DirMaker, error) {
	prefix := filepath.Join(Conf.Worker_dir, system)
	log.Printf("Storage dir at %s", prefix)
	if err := os.RemoveAll(prefix); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(prefix, 0777); err != nil {
		return nil, err
	}

	if mode == STORE_MEMORY {
		// TODO: configure mem size?
		if err := syscall.Mount("none", prefix, "tmpfs", 0, "size=64m"); err != nil {
			return nil, err
		}
	} else if mode == STORE_PRIVATE {
		if err := syscall.Mount(prefix, prefix, "", BIND, ""); err != nil {
			return nil, fmt.Errorf("failed to bind %s: %v", prefix, err)
		}

		if err := syscall.Mount("none", prefix, "", PRIVATE, ""); err != nil {
			return nil, fmt.Errorf("failed to make %s private: %v", prefix, err)
		}
	}

	return &DirMaker{
		prefix: prefix,
		mode:   mode,
	}, nil
}

func (dm *DirMaker) Get(suffix string) string {
	if suffix != "" {
		suffix = "-" + suffix
	}
	id := fmt.Sprintf("%d", atomic.AddInt64(&nextDirId, 1))
	return filepath.Join(dm.prefix, id) + suffix
}

func (dm *DirMaker) Make(suffix string) string {
	dir := dm.Get(suffix)
	if err := os.Mkdir(dir, 0777); err != nil {
		panic(err)
	}
	return dir
}

func (dm *DirMaker) Cleanup() error {
	if dm.mode == STORE_PRIVATE || dm.mode == STORE_MEMORY {
		if err := syscall.Unmount(dm.prefix, syscall.MNT_DETACH); err != nil {
			panic(err)
		}
	}
	return os.RemoveAll(dm.prefix)
}

package sandbox

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/ol/config"
)

/*
#include <sys/eventfd.h>
*/
import "C"

type ImportCacheContainerFactory struct {
	handlerFactory *SOCKPool
	cacheFactory   *SOCKPool
	cacheDir       string
	mutex          sync.Mutex
	servers        []*ForkServer
	seq            int
	full           int32
}

// TODO: make this part of the CacheManager
type Evictor struct {
	ic        *ImportCacheContainerFactory
	limit     int
	eventfd   int
	usagePath string
}

type ForkServer struct {
	sandbox  Sandbox
	Imports  map[string]bool
	Hits     float64
	Parent   *ForkServer
	Children int
	Size     float64
	Mutex    *sync.Mutex
}

func NewImportCacheContainerFactory(handlerFactory, cacheFactory *SOCKPool) (*ImportCacheContainerFactory, error) {
	cacheDir := filepath.Join(config.Conf.Worker_dir, "import-cache")
	if err := os.MkdirAll(cacheDir, os.ModeDir); err != nil {
		return nil, fmt.Errorf("failed to create pool directory at %s :: %v", cacheDir, err)
	}

	ic := &ImportCacheContainerFactory{
		handlerFactory: handlerFactory,
		cacheFactory:   cacheFactory,
		servers:        make([]*ForkServer, 0, 0),
		seq:            0,
		cacheDir:       cacheDir,
	}

	if err := ic.initCacheRoot(); err != nil {
		return nil, err
	}

	_, err := NewEvictor(ic, config.Conf.Import_cache_mb)
	if err != nil {
		return nil, err
	}

	return ic, nil
}

func (ic *ImportCacheContainerFactory) Create(handlerDir, workingDir string, imports []string) (Sandbox, error) {
	parent, err := ic.FindOrMakeParent(imports)
	if err != nil {
		return nil, err
	}

	child, err := ic.handlerFactory.CreateFromParent(handlerDir, workingDir, imports, parent)
	if err != nil {
		return nil, err
	}

	return child, nil
}

func (ic *ImportCacheContainerFactory) FindOrMakeParent(imports []string) (parent Sandbox, err error) {
	ic.mutex.Lock()

	// find parent with greatest import subset
	servers := ic.servers
	best_fs := servers[0]
	best_score := -1
	best_toCache := imports
	for i := 1; i < len(servers); i++ {
		matched := 0
		toCache := make([]string, 0, 0)
		for j := 0; j < len(imports); j++ {
			if servers[i].Imports[imports[j]] {
				matched += 1
			} else {
				toCache = append(toCache, imports[j])
			}
		}

		// constrain to subset
		if matched > best_score && len(servers[i].Imports) <= matched {
			best_fs = servers[i]
			best_score = matched
			best_toCache = toCache
		}
	}

	if best_fs == nil {
		ic.mutex.Unlock()
		return nil, errors.New("no match?")
	}

	// create new, better cache entry
	if len(best_toCache) != 0 && !ic.Full() {
		baseFS := best_fs
		baseFS.Mutex.Lock()
		best_fs, err = ic.newCacheEntry(baseFS, best_toCache)
		baseFS.Mutex.Unlock()

		if err != nil {
			ic.mutex.Unlock()
			return nil, err
		}

		ic.servers = append(ic.servers, best_fs)
		ic.seq++

		// this lock must be taken here to avoid us from being victimized
		best_fs.Mutex.Lock()
		ic.mutex.Unlock()

		best_toCache = []string{}
	} else {
		// we must take this lock to prevent the fork server from being
		// reclaimed while we are waiting to create the container... not ideal
		best_fs.Mutex.Lock()
		ic.mutex.Unlock()
	}

	// keep track of number of hits
	best_fs.Hit()

	// signal interpreter to forkenter into sandbox's namespace
	best_fs.Mutex.Unlock()

	return best_fs.sandbox, nil
}

func (ic *ImportCacheContainerFactory) Cleanup() {
	// TODO: release all containers
	ic.handlerFactory.Cleanup()
	ic.cacheFactory.Cleanup()
}

func (ic *ImportCacheContainerFactory) newCacheEntry(baseFS *ForkServer, toCache []string) (*ForkServer, error) {
	// make hashset of packages for new entry
	var err error
	imports := make(map[string]bool)

	for key, val := range baseFS.Imports {
		imports[key] = val
	}
	for k := 0; k < len(toCache); k++ {
		imports[toCache[k]] = true
	}

	// get container for new entry
	sandbox, err := ic.cacheFactory.CreateFromParent("", ic.cacheDir, []string{}, nil)
	if err != nil {
		return nil, err
	}

	fs := &ForkServer{
		Imports:  imports,
		Hits:     0.0,
		Parent:   baseFS,
		Children: 0,
		Mutex:    &sync.Mutex{},
		sandbox:  sandbox,
	}

	baseFS.Children += 1

	// signal interpreter to forkenter into sandbox's namespace
	err = baseFS.sandbox.fork(sandbox, toCache, false)
	if err != nil {
		fs.Kill()
		return nil, err
	}

	return fs, nil
}

func (ic *ImportCacheContainerFactory) initCacheRoot() (err error) {
	rootSB, err := ic.cacheFactory.CreateFromParent("", ic.cacheDir, []string{}, nil)
	if err != nil {
		return fmt.Errorf("failed to create root cache entry :: %v", err)
	}

	fs := &ForkServer{
		sandbox:  rootSB,
		Imports:  make(map[string]bool),
		Hits:     0.0,
		Parent:   nil,
		Children: 0,
		Mutex:    &sync.Mutex{},
		Size:     1.0, // divide-by-zero
	}

	ic.servers = append(ic.servers, fs)

	return nil
}

func (ic *ImportCacheContainerFactory) Full() bool {
	return atomic.LoadInt32(&ic.full) == 1
}

func (ic *ImportCacheContainerFactory) PrintDebug() {
	fmt.Printf("CACHE SANDBOXES:\n\n")
	ic.cacheFactory.PrintDebug()
	fmt.Printf("HANDLER SANDBOXES:\n\n")
	ic.handlerFactory.PrintDebug()
}

func NewEvictor(ic *ImportCacheContainerFactory, mb_limit int) (*Evictor, error) {
	byte_limit := mb_limit * 1024 * 1024

	eventfd, err := C.eventfd(0, C.EFD_CLOEXEC)
	if err != nil {
		return nil, err
	}

	usagePath := filepath.Join(ic.cacheFactory.cgPool.Path("memory"), "memory.usage_in_bytes")
	usagefd, err := syscall.Open(usagePath, syscall.O_RDONLY, 0777)
	if err != nil {
		return nil, fmt.Errorf("could not open %s :: %s", usagePath, err)
	}

	// TODO: do we even use the eventfd?
	eventPath := filepath.Join(ic.cacheFactory.cgPool.Path("memory"), "cgroup.event_control")
	eventStr := fmt.Sprintf("'%d %d %d'", eventfd, usagefd, byte_limit)
	echo := exec.Command("echo", eventStr, ">", eventPath)
	if err = echo.Run(); err != nil {
		return nil, fmt.Errorf("could not write to %s :: %s", eventPath, err)
	}

	e := &Evictor{
		ic:        ic,
		limit:     byte_limit,
		eventfd:   int(eventfd),
		usagePath: usagePath,
	}

	go e.EvictorTask()

	return e, nil
}

func (e *Evictor) EvictorTask() {
	for {
		time.Sleep(50 * time.Millisecond)
		e.ic.mutex.Lock()

		usage := e.usage()
		if usage > e.limit {
			atomic.StoreInt32(&e.ic.full, 1)
			e.evict()
		} else {
			atomic.StoreInt32(&e.ic.full, 0)
		}

		e.ic.mutex.Unlock()
	}
}

func (e *Evictor) usage() (usage int) {
	buf, err := ioutil.ReadFile(e.usagePath)
	if err != nil {
		return 0
	}

	str := strings.TrimSpace(string(buf[:]))
	usage, err = strconv.Atoi(str)
	if err != nil {
		panic(fmt.Sprintf("atoi failed: %v", err))
	}

	return usage
}

func (e *Evictor) evict() {
	servers := e.ic.servers
	idx := -1
	worst := float64(math.Inf(+1))

	for k := 1; k < len(servers); k++ {
		if servers[k].Children == 0 {
			if ratio := servers[k].Hits / servers[k].Size; ratio < worst {
				idx = k
				worst = ratio
			}
		}
	}

	if idx != -1 {
		// TODO: make sure no one else is using this one
		victim := servers[idx]
		e.ic.servers = append(servers[:idx], servers[idx+1:]...)
		go victim.Kill()
	} else {
		log.Printf("No victim found")
	}
}

func (fs *ForkServer) Hit() {
	curr := fs
	for curr != nil {
		curr.Hits += 1.0
		curr = curr.Parent
	}
}

func (fs *ForkServer) Kill() error {
	if fs.Parent != nil {
		fs.Parent.Children -= 1
	}

	fs.sandbox.Destroy()

	return nil
}

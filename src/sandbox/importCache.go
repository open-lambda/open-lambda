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
	handlerFactory *SOCKContainerFactory
	cacheFactory   *SOCKContainerFactory
	*CacheManager
}

type CacheFactory struct {
	delegate ContainerFactory
	cacheDir string
}

type CacheManager struct {
	factory *CacheFactory
	servers []*ForkServer
	seq     int
	mutex   *sync.Mutex
	full    *int32
}

// TODO: make this part of the CacheManager
type Evictor struct {
	cm        *CacheManager
	limit     int
	eventfd   int
	usagePath string
}

type ForkServer struct {
	sandbox  *SOCKContainer
	Imports  map[string]bool
	Hits     float64
	Parent   *ForkServer
	Children int
	Size     float64
	Mutex    *sync.Mutex
	Dead     bool
	Pipe     *os.File
}

func NewImportCacheContainerFactory(handlerFactory, cacheFactory *SOCKContainerFactory) (*ImportCacheContainerFactory, error) {
	cacheMgr, err := NewCacheManager(cacheFactory)
	if err != nil {
		return nil, err
	}

	return &ImportCacheContainerFactory{
		handlerFactory: handlerFactory,
		cacheFactory:   cacheFactory,
		CacheManager:   cacheMgr,
	}, nil
}

func (ic *ImportCacheContainerFactory) Create(handlerDir, workingDir string, imports []string) (Sandbox, error) {
	parent, err := ic.CacheManager.FindOrMakeParent(imports)
	if err != nil {
		return nil, err
	}

	child, err := ic.handlerFactory.CreateFromParent(handlerDir, workingDir, imports, parent)
	if err != nil {
		return nil, err
	}

	return child, nil
}

func (ic *ImportCacheContainerFactory) Cleanup() {
	ic.handlerFactory.Cleanup()
	ic.cacheFactory.Cleanup()
}

func (cf *CacheFactory) Create() (*SOCKContainer, error) {
	c, err := cf.delegate.Create("", cf.cacheDir, []string{})
	return c.(*SOCKContainer), err
}

func (cf *CacheFactory) Cleanup() {
	cf.delegate.Cleanup()
}

func NewCacheManager(cacheFactory *SOCKContainerFactory) (cm *CacheManager, err error) {
	servers := make([]*ForkServer, 0, 0)
	if err != nil {
		return nil, err
	}

	var full int32 = 0
	cm = &CacheManager{
		servers: servers,
		seq:     0,
		mutex:   &sync.Mutex{},
		full:    &full,
	}

	memCGroupPath, err := cm.initCacheRoot(cacheFactory)
	if err != nil {
		return nil, err
	}

	e, err := NewEvictor(cm, "", memCGroupPath, config.Conf.Import_cache_mb)
	if err != nil {
		return nil, err
	}

	go func(cm *CacheManager) {
		for {
			time.Sleep(50 * time.Millisecond)
			e.CheckUsage()
		}
	}(cm)

	return cm, nil
}

func (cm *CacheManager) FindOrMakeParent(imports []string) (parent *SOCKContainer, err error) {
	cm.mutex.Lock()

	// find parent with greatest import subset
	servers := cm.servers
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
		cm.mutex.Unlock()
		return nil, errors.New("no match?")
	}

	// create new, better cache entry
	if len(best_toCache) != 0 && !cm.Full() {
		baseFS := best_fs
		baseFS.Mutex.Lock()
		best_fs, err = cm.newCacheEntry(baseFS, best_toCache)
		baseFS.Mutex.Unlock()

		if err != nil {
			cm.mutex.Unlock()
			return nil, err
		}

		cm.servers = append(cm.servers, best_fs)
		cm.seq++

		// this lock must be taken here to avoid us from being victimized
		best_fs.Mutex.Lock()
		cm.mutex.Unlock()

		if err := best_fs.WaitForEntryInit(); err != nil {
			return nil, err
		}

		best_toCache = []string{}
	} else {
		// we must take this lock to prevent the fork server from being
		// reclaimed while we are waiting to create the container... not ideal
		best_fs.Mutex.Lock()
		cm.mutex.Unlock()
	}

	// keep track of number of hits
	best_fs.Hit()

	// signal interpreter to forkenter into sandbox's namespace
	best_fs.Mutex.Unlock()

	return best_fs.sandbox, nil
}

func (cm *CacheManager) newCacheEntry(baseFS *ForkServer, toCache []string) (*ForkServer, error) {
	// make hashset of packages for new entry
	var err error
	imports := make(map[string]bool)

	for key, val := range baseFS.Imports {
		imports[key] = val
	}
	for k := 0; k < len(toCache); k++ {
		imports[toCache[k]] = true
	}

	fs := &ForkServer{
		Imports:  imports,
		Hits:     0.0,
		Parent:   baseFS,
		Children: 0,
		Mutex:    &sync.Mutex{},
	}

	baseFS.Children += 1

	// get container for new entry
	sandbox, err := cm.factory.Create()
	if err != nil {
		fs.Kill()
		return nil, err
	}

	// open pipe before forkenter
	pipeDir := filepath.Join(sandbox.HostDir(), "server_pipe")
	fs.Pipe, err = os.OpenFile(pipeDir, os.O_RDWR, 0777)
	if err != nil {
		log.Fatalf("Cannot open pipe: %v\n", err)
	}

	// signal interpreter to forkenter into sandbox's namespace
	err = baseFS.sandbox.Fork(sandbox, toCache, false)
	if err != nil {
		fs.Kill()
		return nil, err
	}

	fs.sandbox = sandbox

	return fs, nil
}

func (cm *CacheManager) initCacheRoot(cacheFactory *SOCKContainerFactory) (memCGroupPath string, err error) {
	cacheDir := filepath.Join(config.Conf.Worker_dir, "import-cache")
	if err := os.MkdirAll(cacheDir, os.ModeDir); err != nil {
		return "", fmt.Errorf("failed to create pool directory at %s :: %v", cacheDir, err)
	}

	cm.factory = &CacheFactory{cacheFactory, cacheDir}

	rootSB, err := cm.factory.Create()
	if err != nil {
		return "", fmt.Errorf("failed to create root cache entry :: %v", err)
	}

	// open pipe before forkenter
	pipeDir := filepath.Join(rootSB.HostDir(), "server_pipe")
	pipe, err := os.OpenFile(pipeDir, os.O_RDWR, 0777)
	if err != nil {
		log.Fatalf("Cannot open pipe: %v\n", err)
	}

	start := time.Now()
	// use StdoutPipe of olcontainer to sync with lambda server
	ready := make(chan bool, 1)
	defer close(ready)
	go func() {
		defer pipe.Close()

		// wait for "ready"
		buf := make([]byte, 5)
		_, err := pipe.Read(buf)
		if err != nil {
			log.Printf("Cannot read from stdout of olcontainer: %v\n", err)
		} else if string(buf) != "ready" {
			log.Printf("Expect to read `ready`, only found %s\n", string(buf))
		}
		ready <- true
	}()

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	start = time.Now()
	select {
	case <-ready:
		if config.Conf.Timing {
			log.Printf("wait for server took %v\n", time.Since(start))
		}
	case <-timeout.C:
		if n, err := pipe.Write([]byte("timeo")); err != nil {
			return "", err
		} else if n != 5 {
			return "", fmt.Errorf("Cannot write `timeo` to pipe\n")
		}
		return "", errors.New("root forkserver failed to start after 5s")
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

	cm.servers = append(cm.servers, fs)

	return rootSB.cg.Path("memory", ""), nil
}

func (cm *CacheManager) Full() bool {
	return atomic.LoadInt32(cm.full) == 1
}

func NewEvictor(cm *CacheManager, pkgfile, memCGroupPath string, mb_limit int) (*Evictor, error) {
	byte_limit := mb_limit * 1024 * 1024

	eventfd, err := C.eventfd(0, C.EFD_CLOEXEC)
	if err != nil {
		return nil, err
	}

	usagePath := filepath.Join(memCGroupPath, "memory.usage_in_bytes")
	usagefd, err := syscall.Open(usagePath, syscall.O_RDONLY, 0777)
	if err != nil {
		return nil, fmt.Errorf("could not open %s :: %s", usagePath, err)
	}

	eventPath := filepath.Join(memCGroupPath, "cgroup.event_control")

	eventStr := fmt.Sprintf("'%d %d %d'", eventfd, usagefd, byte_limit)
	echo := exec.Command("echo", eventStr, ">", eventPath)
	if err = echo.Run(); err != nil {
		return nil, fmt.Errorf("could not write to %s :: %s", eventPath, err)
	}

	e := &Evictor{
		cm:        cm,
		limit:     byte_limit,
		eventfd:   int(eventfd),
		usagePath: usagePath,
	}

	return e, nil
}

func (e *Evictor) CheckUsage() {
	e.cm.mutex.Lock()
	defer e.cm.mutex.Unlock()

	usage := e.usage()
	if usage > e.limit {
		atomic.StoreInt32(e.cm.full, 1)
		e.evict()
	} else {
		atomic.StoreInt32(e.cm.full, 0)
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
	servers := e.cm.servers
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
		e.cm.servers = append(servers[:idx], servers[idx+1:]...)
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
	fs.Dead = true

	if fs.Parent != nil {
		fs.Parent.Children -= 1
	}

	fs.sandbox.Destroy()

	return nil
}

func (fs *ForkServer) WaitForEntryInit() error {
	// use StdoutPipe of olcontainer to sync with lambda server
	ready := make(chan bool, 1)
	defer close(ready)
	go func() {
		defer fs.Pipe.Close()

		// wait for "ready"
		buf := make([]byte, 5)
		_, err := fs.Pipe.Read(buf)
		if err != nil {
			log.Printf("Cannot read from stdout of olcontainer: %v\n", err)
		} else if string(buf) != "ready" {
			log.Printf("Expect to read `ready`, but found %v\n", string(buf))
		} else {
			ready <- true
		}
	}()

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	start := time.Now()
	select {
	case <-ready:
		log.Printf("wait for server took %v\n", time.Since(start))
	case <-timeout.C:
		if n, err := fs.Pipe.Write([]byte("timeo")); err != nil {
			return err
		} else if n != 5 {
			return fmt.Errorf("Cannot write `timeo` to pipe\n")
		}
		return fmt.Errorf("Cache entry failed to initialize after 5s")
	}

	return nil
}

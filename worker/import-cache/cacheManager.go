package cache

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sb "github.com/open-lambda/open-lambda/worker/sandbox"

	"github.com/open-lambda/open-lambda/worker/config"
)

type CacheManager struct {
	factory CacheFactory
	servers []*ForkServer
	matcher CacheMatcher
	seq     int
	mutex   *sync.Mutex
	sizes   map[string]float64
	full    *int32
}

func InitCacheManager(opts *config.Config) (cm *CacheManager, err error) {
	if opts.Import_cache_size == 0 {
		return nil, nil
	}

	servers := make([]*ForkServer, 0, 0)
	sizes, err := readPkgSizes("/ol/open-lambda/worker/cache-manager/package_sizes.txt")
	if err != nil {
		return nil, err
	}

	var full int32 = 0
	cm = &CacheManager{
		servers: servers,
		matcher: NewSubsetMatcher(),
		seq:     0,
		mutex:   &sync.Mutex{},
		sizes:   sizes,
		full:    &full,
	}

	memCGroupPath, err := cm.initCacheRoot(opts)
	if err != nil {
		return nil, err
	}

	e, err := NewEvictor(cm, "", memCGroupPath, opts.Import_cache_size)
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

func (cm *CacheManager) Provision(sandbox sb.Container, imports []string) (fs *ForkServer, hit bool, err error) {
	if config.Timing {
		defer func(start time.Time) {
			log.Printf("provision took %v\n", time.Since(start))
		}(time.Now())
	}

	cm.mutex.Lock()

	fs, toCache, hit := cm.matcher.Match(cm.servers, imports)
	if fs == nil {
		cm.mutex.Unlock()
		return nil, false, errors.New("no match?")
	}

	if len(toCache) != 0 && !cm.Full() {
		baseFS := fs
		baseFS.Mutex.Lock()
		fs, err = cm.newCacheEntry(baseFS, toCache)
		baseFS.Mutex.Unlock()

		if err != nil {
			cm.mutex.Unlock()
			return nil, false, err
		}

		cm.servers = append(cm.servers, fs)
		cm.seq++

		// this lock must be taken here to avoid us from being victimized
		fs.Mutex.Lock()
		cm.mutex.Unlock()

		if err := fs.WaitForEntryInit(); err != nil {
			return nil, false, err
		}

		toCache = []string{}
	} else {
		// we must take this lock to prevent the fork server from being
		// reclaimed while we are waiting to create the container... not ideal
		fs.Mutex.Lock()
		cm.mutex.Unlock()
	}

	// keep track of number of hits
	fs.Hit()

	// signal interpreter to forkenter into sandbox's namespace
	pid, err := forkRequest(fs.SockPath, sandbox.NSPid(), sandbox.RootDir(), toCache, true)
	if err != nil {
		fs.Mutex.Unlock()
		return nil, false, err
	}

	fs.Mutex.Unlock()

	// change cgroup of spawned lambda server
	if err = sandbox.CGroupEnter(pid); err != nil {
		return nil, false, err
	}

	return fs, hit, nil
}

func (cm *CacheManager) newCacheEntry(baseFS *ForkServer, toCache []string) (*ForkServer, error) {
	// make hashset of packages for new entry
	var err error
	imports := make(map[string]bool)
	size := 0.0
	for key, val := range baseFS.Imports {
		imports[key] = val
	}
	for k := 0; k < len(toCache); k++ {
		imports[toCache[k]] = true
		size += cm.sizes[strings.ToLower(toCache[k])]
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
	pid, err := forkRequest(baseFS.SockPath, sandbox.NSPid(), sandbox.RootDir(), toCache, false)
	if err != nil {
		fs.Kill()
		return nil, err
	}

	fs.Sandbox = sandbox
	fs.Pid = pid
	fs.SockPath = fmt.Sprintf("%s/fs.sock", sandbox.HostDir())

	return fs, nil
}

func (cm *CacheManager) initCacheRoot(opts *config.Config) (memCGroupPath string, err error) {
	factory, rootSB, rootDir, err := NewCacheFactory(opts)
	if err != nil {
		return "", err
	}
	cm.factory = factory

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
		if opts.Timing {
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
		Sandbox:  rootSB,
		Pid:      rootSB.NSPid(),
		SockPath: fmt.Sprintf("%s/fs.sock", rootDir),
		Imports:  make(map[string]bool),
		Hits:     0.0,
		Parent:   nil,
		Children: 0,
		Mutex:    &sync.Mutex{},
		Size:     1.0, // divide-by-zero
	}

	cm.servers = append(cm.servers, fs)

	return rootSB.MemoryCGroupPath(), nil
}

func (cm *CacheManager) Full() bool {
	return atomic.LoadInt32(cm.full) == 1
}

func readPkgSizes(path string) (map[string]float64, error) {
	sizes := make(map[string]float64)
	file, err := os.Open(path)
	if err != nil {
		log.Printf("invalid package sizes path %v, using 0 for all", path)
		return make(map[string]float64), nil
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if err = scanner.Err(); err != nil {
			return nil, err
		}

		split := strings.Split(scanner.Text(), ":")
		if len(split) != 2 {
			return nil, errors.New("malformed package size file")
		}

		size, err := strconv.Atoi(split[1])
		if err != nil {
			return nil, err
		}
		sizes[strings.ToLower(split[1])] = float64(size)
	}

	return sizes, nil
}

func (cm *CacheManager) Cleanup() {
	for _, server := range cm.servers {
		server.Kill()
	}

	cm.factory.Cleanup()
}

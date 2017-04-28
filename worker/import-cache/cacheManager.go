package cache

import (
	"log"
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sb "github.com/open-lambda/open-lambda/worker/sandbox"

	"github.com/open-lambda/open-lambda/worker/config"
)

type CacheManager struct {
	factory *BufferedCacheFactory
	cluster string
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
		cluster: opts.Cluster_name,
		servers: servers,
		matcher: NewSubsetMatcher(),
		seq:     0,
		mutex:   &sync.Mutex{},
		sizes:   sizes,
		full:    &full,
	}

	rootCID, err := cm.initCacheRoot(opts.Import_cache_dir, opts.Pkgs_dir, opts.Import_cache_buffer)
	if err != nil {
		return nil, err
	}

	e, err := NewEvictor("", rootCID, opts.Import_cache_size)
	if err != nil {
		return nil, err
	}

	go func(cm *CacheManager) {
		for {
			time.Sleep(50 * time.Millisecond)
			cm.servers = e.CheckUsage(cm.servers, cm.mutex, cm.full)
		}
	}(cm)

	return cm, nil
}

func (cm *CacheManager) Provision(sandbox sb.ContainerSandbox, dir string, pkgs []string) (fs *ForkServer, hit bool, err error) {
	cm.mutex.Lock()

	fs, toCache, hit := cm.matcher.Match(cm.servers, pkgs)

	// make new cache entry if necessary
	if len(toCache) != 0 {
		fs, err = cm.newCacheEntry(fs, toCache)
		if err != nil {
			return nil, false, err
			//return cm.Provision(sandbox, dir, pkgs) //TODO
		}
	} else {
		cm.mutex.Unlock()
		fs.Mutex.Lock()
		if fs == nil {
			return nil, false, err
			//return cm.Provision(sandbox, dir, pkgs) //TODO
		}
	}
	defer fs.Mutex.Unlock()

	// keep track of number of hits
	fs.Hit()

	// signal interpreter to forkenter into sandbox's namespace
	pid, err := forkRequest(fs.SockPath, sandbox.NSPid(), []string{}, true)
	if err != nil {
		return nil, false, err
	}

	// change cgroup of spawned lambda server
	if err = sandbox.CGroupEnter(pid); err != nil {
		return nil, false, err
	}

	return fs, hit, nil
}

func (cm *CacheManager) newCacheEntry(fs *ForkServer, toCache []string) (*ForkServer, error) {
	// make hashset of packages for new entry
	pkgs := make(map[string]bool)
	size := 0.0
	for key, val := range fs.Packages {
		pkgs[key] = val
	}
	for k := 0; k < len(toCache); k++ {
		pkgs[toCache[k]] = true
		size += cm.sizes[strings.ToLower(toCache[k])]
	}

	newFs := &ForkServer{
		Packages: pkgs,
		Hits:     0.0,
		Parent:   fs,
		Children: 0,
		Mutex:    &sync.Mutex{},
	}

	fs.Children += 1

	cm.servers = append(cm.servers, newFs)
	cm.seq++

	newFs.Mutex.Lock()
	cm.mutex.Unlock()

	// get container for new entry
	sandbox, dir, err := cm.factory.Create()
	if err != nil {
		newFs.Kill()
		return nil, err
	}

	// signal interpreter to forkenter into sandbox's namespace
	pid, err := forkRequest(fs.SockPath, sandbox.NSPid(), toCache, false)
	if err != nil {
		newFs.Kill()
		return nil, err
	}

	sockPath := fmt.Sprintf("%s/fs.sock", dir)

	// wait up to 30s for server to initialize
	start := time.Now()
	for ok := true; ok; ok = os.IsNotExist(err) {
		_, err = os.Stat(sockPath)
		if time.Since(start).Seconds() > 500 {
			return nil, errors.New(fmt.Sprintf("cache server %d failed to initialize after 500s", cm.seq))
		}
		time.Sleep(1 * time.Millisecond)
	}

	newFs.Sandbox = sandbox
	newFs.Pid = pid
	newFs.SockPath = sockPath

	return newFs, nil
}

func (cm *CacheManager) initCacheRoot(poolDir, pkgsDir string, buffer int) (rootCID string, err error) {
	factory, rootSB, rootDir, rootCID, err := InitCacheFactory(poolDir, pkgsDir, cm.cluster, buffer)
	if err != nil {
		return "", err
	}
	cm.factory = factory

	// wait up to 5s for root server to spawn
	pidPath := fmt.Sprintf("%s/pid", rootDir)
	start := time.Now()
	for ok := true; ok; ok = os.IsNotExist(err) {
		_, err = os.Stat(pidPath)
		if time.Since(start).Seconds() > 5 {
			return "", errors.New("root forkserver failed to start after 5s")
		}
	}

	pidFile, err := os.Open(pidPath)
	if err != nil {
		return "", err
	}
	defer pidFile.Close()

	scnr := bufio.NewScanner(pidFile)
	scnr.Scan()
	pid := scnr.Text()

	if err := scnr.Err(); err != nil {
		return "", err
	}

	fs := &ForkServer{
		Sandbox:  rootSB,
		Pid:      pid,
		SockPath: fmt.Sprintf("%s/fs.sock", rootDir),
		Packages: make(map[string]bool),
		Hits:     0.0,
		Parent:   nil,
		Children: 0,
		Mutex:    &sync.Mutex{},
		Size:     1.0, // divide-by-zero
	}

	cm.servers = append(cm.servers, fs)

	return rootCID, nil
}

func (cm *CacheManager) Full() bool {
	return atomic.LoadInt32(cm.full) == 1
}

func readPkgSizes(path string) (map[string]float64, error) {
	sizes := make(map[string]float64)
	file, err := os.Open(path)
	if err != nil {
		log.Printf("invalid package sizes path %v, using 0 for all", path);
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

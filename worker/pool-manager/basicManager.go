package pmanager

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	sb "github.com/open-lambda/open-lambda/worker/sandbox"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/pool-manager/policy"
)

type BasicManager struct {
	factory *BufferedCacheFactory
	cluster string
	servers []*policy.ForkServer
	matcher policy.CacheMatcher
	seq     int
	mutex   *sync.Mutex
	sizes   map[string]float64
}

func NewBasicManager(opts *config.Config) (bm *BasicManager, err error) {
	servers := make([]*policy.ForkServer, 0, 0)
	sizes, err := readPkgSizes("/ol/open-lambda/worker/pool-manager/package_sizes.txt")
	if err != nil {
		return nil, err
	}

	bm = &BasicManager{
		cluster: opts.Cluster_name,
		servers: servers,
		matcher: policy.NewSubsetMatcher(),
		seq:     0,
		mutex:   &sync.Mutex{},
		sizes:   sizes,
	}

	rootCID, err := bm.initCacheRoot(opts.Import_cache_dir, opts.Pkgs_dir, opts.Import_cache_buffer)
	if err != nil {
		return nil, err
	}

	e, err := policy.NewEvictor("", rootCID, opts.Import_cache_size)
	if err != nil {
		return nil, err
	}

	go func(bm *BasicManager) {
		for {
			time.Sleep(50 * time.Millisecond)
			bm.servers = e.CheckUsage(bm.servers, bm.mutex)
		}
	}(bm)

	return bm, nil
}

func (bm *BasicManager) Provision(sandbox sb.ContainerSandbox, dir string, pkgs []string) (fs *policy.ForkServer, err error) {
	bm.mutex.Lock()

	fs, toCache := bm.matcher.Match(bm.servers, pkgs)

	// make new cache entry if necessary
	if len(toCache) != 0 {
		fs, err = bm.newCacheEntry(fs, toCache)
		if err != nil {
			return nil, err
		}
	} else {
		bm.mutex.Unlock()
		fs.Mutex.Lock()
	}
	defer fs.Mutex.Unlock()

	// keep track of number of hits
	fs.Hit()

	// signal interpreter to forkenter into sandbox's namespace
	pid, err := forkRequest(fs.SockPath, sandbox.NSPid(), []string{}, true)
	if err != nil {
		return nil, err
	}

	// change cgroup of spawned lambda server
	if err = sandbox.CGroupEnter(pid); err != nil {
		return nil, err
	}

	return fs, nil
}

func (bm *BasicManager) newCacheEntry(fs *policy.ForkServer, toCache []string) (*policy.ForkServer, error) {
	// make hashset of packages for new entry
	pkgs := make(map[string]bool)
	size := 0.0
	for key, val := range fs.Packages {
		pkgs[key] = val
	}
	for k := 0; k < len(toCache); k++ {
		pkgs[toCache[k]] = true
		size += bm.sizes[strings.ToLower(toCache[k])]
	}

	newFs := &policy.ForkServer{
		Packages: pkgs,
		Hits:     0.0,
		Parent:   fs,
		Children: 0,
		Mutex:    &sync.Mutex{},
	}

	fs.Children += 1

	bm.servers = append(bm.servers, newFs)
	bm.seq++

	newFs.Mutex.Lock()
	bm.mutex.Unlock()

	// get container for new entry
	sandbox, dir, err := bm.factory.Create()
	if err != nil {
		return nil, err
	}

	// signal interpreter to forkenter into sandbox's namespace
	pid, err := forkRequest(fs.SockPath, sandbox.NSPid(), toCache, false)
	if err != nil {
		return nil, err
	}

	sockPath := fmt.Sprintf("%s/fs.sock", dir)

	// wait up to 30s for server to initialize
	start := time.Now()
	for ok := true; ok; ok = os.IsNotExist(err) {
		_, err = os.Stat(sockPath)
		if time.Since(start).Seconds() > 30 {
			return nil, errors.New(fmt.Sprintf("cache server %d failed to initialize after 30s", bm.seq))
		}
	}

	newFs.Sandbox = sandbox
	newFs.Pid = pid
	newFs.SockPath = sockPath

	return newFs, nil
}

func (bm *BasicManager) initCacheRoot(poolDir, pkgsDir string, buffer int) (rootCID string, err error) {
	factory, rootSB, rootDir, rootCID, err := InitCacheFactory(poolDir, pkgsDir, bm.cluster, buffer)
	if err != nil {
		return "", err
	}
	bm.factory = factory

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

	fs := &policy.ForkServer{
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

	bm.servers = append(bm.servers, fs)

	return rootCID, nil
}

func readPkgSizes(path string) (map[string]float64, error) {
	sizes := make(map[string]float64)
	file, err := os.Open(path)
	if err != nil {
		return nil, err
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

package pmanager

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	sb "github.com/open-lambda/open-lambda/worker/sandbox"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/pool-manager/policy"
)

type BasicManager struct {
	factory *BufferedCacheFactory
	cluster string
	servers []policy.ForkServer
	matcher policy.CacheMatcher
	seq     int
	mutex   sync.Mutex
}

func NewBasicManager(opts *config.Config) (bm *BasicManager, err error) {
	servers := make([]policy.ForkServer, 0, 0)
	bm = &BasicManager{
		cluster: opts.Cluster_name,
		servers: servers,
		matcher: policy.NewSubsetMatcher(),
		seq:     0,
	}

	rootCID, err := bm.initCacheRoot(opts.Pool_dir)
	if err != nil {
		return nil, err
	}

	e, err := policy.NewEvictor("", rootCID, 1000)
	if err != nil {
		return nil, err
	}

	go func(bm *BasicManager) {
		for {
			time.Sleep(50 * time.Millisecond)
			bm.servers = e.CheckUsage(bm.servers, &bm.mutex)
		}
	}(bm)

	return bm, nil
}

func (bm *BasicManager) Provision(sandbox sb.ContainerSandbox, dir string, pkgs []string) (err error) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	fs, toCache := bm.matcher.Match(bm.servers, pkgs)

	// make new cache entry if necessary
	if len(toCache) != 0 {
		fs, err = bm.newCacheEntry(fs, toCache)
		if err != nil {
			return err
		}
	}

	// keep track of number of hits
	fs.Hit()

	// signal interpreter to forkenter into sandbox's namespace
	pid, err := forkRequest(fs.SockPath, sandbox.NSPid(), []string{}, true)
	if err != nil {
		return err
	}

	// change cgroup of spawned lambda server
	if err = sandbox.CGroupEnter(pid); err != nil {
		return err
	}

	sockPath := fmt.Sprintf("%s/ol.sock", dir)

	// wait up to 15s for server to initialize
	start := time.Now()
	for ok := true; ok; ok = os.IsNotExist(err) {
		_, err = os.Stat(sockPath)
		if time.Since(start).Seconds() > 10 {
			return errors.New(fmt.Sprintf("handler server failed to initialize after 10s"))
		}
	}

	return nil
}

func (bm *BasicManager) newCacheEntry(fs *policy.ForkServer, toCache []string) (*policy.ForkServer, error) {
	// make hashset of packages for new entry
	pkgs := make(map[string]bool)
	for key, val := range fs.Packages {
		pkgs[key] = val
	}
	for k := 0; k < len(toCache); k++ {
		pkgs[toCache[k]] = true
	}

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

	// wait up to 15s for server to initialize
	start := time.Now()
	for ok := true; ok; ok = os.IsNotExist(err) {
		_, err = os.Stat(sockPath)
		if time.Since(start).Seconds() > 30 {
			return nil, errors.New(fmt.Sprintf("cache server %d failed to initialize after 30s", bm.seq))
		}
	}

	newFs := policy.ForkServer{
		Sandbox:  sandbox,
		Pid:      pid,
		SockPath: sockPath,
		Packages: pkgs,
		Hits:     0,
		Parent:   fs,
		Children: 0,
	}

	fs.Children += 1

	bm.servers = append(bm.servers, newFs)
	bm.seq++

	return &newFs, nil
}

func (bm *BasicManager) initCacheRoot(poolDir string) (rootCID string, err error) {
	factory, rootSB, rootDir, rootCID, err := InitCacheFactory(poolDir, bm.cluster, 2) //TODO: buffer
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

	fs := policy.ForkServer{
		Sandbox:  rootSB,
		Pid:      pid,
		SockPath: fmt.Sprintf("%s/fs.sock", rootDir),
		Packages: make(map[string]bool),
		Hits:     0,
		Parent:   nil,
		Children: 0,
	}

	bm.servers = append(bm.servers, fs)

	return rootCID, nil
}

package pmanager

import (
	"bufio"
	"errors"
	"fmt"
	"os"
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
	evictor policy.CacheEvictor
	seq     int
}

func NewBasicManager(opts *config.Config) (bm *BasicManager, err error) {
	servers := make([]policy.ForkServer, 0, 0)
	bm = &BasicManager{
		cluster: opts.Cluster_name,
		servers: servers,
		matcher: policy.NewSubsetMatcher(),
		evictor: policy.NewRandomEvictor(),
		seq:     0,
	}

	if err = bm.initCacheRoot(opts.Pool_dir); err != nil {
		return nil, err
	}

	return bm, nil
}

func (bm *BasicManager) Provision(sandbox sb.ContainerSandbox, pkgs []string) (err error) {
	fs, toCache := bm.matcher.Match(bm.servers, pkgs)

	// make new cache entry if necessary
	if len(toCache) != 0 {
		fs, err = bm.newCacheEntry(fs, toCache)
		if err != nil {
			return err
		}
	}

	// signal interpreter to forkenter into sandbox's namespace
	pid, err := forkRequest(fs.SockPath, sandbox.NSPid(), []string{}, true)
	if err != nil {
		return err
	}

	// change cgroup of spawned lambda server
	if err = sandbox.CGroupEnter(pid); err != nil {
		return err
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

	// wait up to 5s for server to initialize
	start := time.Now()
	for ok := true; ok; ok = os.IsNotExist(err) {
		_, err = os.Stat(sockPath)
		if time.Since(start).Seconds() > 5 {
			return nil, errors.New(fmt.Sprintf("forkserver %d failed to initialize after 5s", bm.seq))
		}
	}

	// TODO: cache entry cgroups?

	newFs := policy.ForkServer{
		Sandbox:  sandbox,
		Pid:      pid,
		SockPath: sockPath,
		Packages: pkgs,
	}

	bm.servers = append(bm.servers, newFs)
	bm.seq++

	return &newFs, nil
}

func (bm *BasicManager) initCacheRoot(poolDir string) (err error) {
	factory, rootSB, rootDir, err := InitCacheFactory(poolDir, bm.cluster, 2) //TODO: buffer
	if err != nil {
		return err
	}
	bm.factory = factory

	// wait up to 5s for root server to spawn
	pidPath := fmt.Sprintf("%s/pid", rootDir)
	start := time.Now()
	for ok := true; ok; ok = os.IsNotExist(err) {
		_, err = os.Stat(pidPath)
		if time.Since(start).Seconds() > 5 {
			return errors.New("root forkserver failed to start after 5s")
		}
	}

	pidFile, err := os.Open(pidPath)
	if err != nil {
		return err
	}
	defer pidFile.Close()

	scnr := bufio.NewScanner(pidFile)
	scnr.Scan()
	pid := scnr.Text()

	if err := scnr.Err(); err != nil {
		return err
	}

	fs := policy.ForkServer{
		Sandbox:  rootSB,
		Pid:      pid,
		SockPath: fmt.Sprintf("%s/fs.sock", rootDir),
		Packages: make(map[string]bool),
	}

	bm.servers = append(bm.servers, fs)

	return nil
}

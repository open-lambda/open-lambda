// handler package implements a library for handling run lambda requests from
// the worker server.
package handler

import (
	"container/list"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler/state"
	"github.com/open-lambda/open-lambda/worker/import-cache"
	"github.com/open-lambda/open-lambda/worker/pip-manager"

	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

// Organizes all lambda functions (the code) and lambda instances (that serve events)
type LambdaMgr struct {
	mutex      sync.Mutex
	lfuncMap   map[string]*LambdaFunc
	codePuller     *CodePuller
	pipMgr     pip.InstallManager
	sbFactory  sb.ContainerFactory
	cacheMgr   *cache.CacheManager
	config     *config.Config
	lru        *LambdaInstanceLRU
	workerDir  string
	maxRunners int
	hhits      *int64
	ihits      *int64
	misses     *int64
}

// Represents a single lambda function (the code)
type LambdaFunc struct {
	name         string
	mutex        sync.Mutex
	lmgr         *LambdaMgr
	instances    *list.List
	listEl       map[*LambdaInstance]*list.Element
	workingDir   string
	maxInstances int
	lastPull     *time.Time
	code         []byte
	codeDir      string
	imports      []string
	installs     []string
}

// Wraps a sandbox that runs a process that can handle lambda events
type LambdaInstance struct {
	name    string
	id      string
	mutex   sync.Mutex
	lfunc   *LambdaFunc
	sandbox sb.Container
	fs      *cache.ForkServer
	hostDir string
	runners int
	usage   int
}

func NewLambdaMgr(opts *config.Config) (mgr *LambdaMgr, err error) {
	var t time.Time

	t = time.Now()
	cp, err := NewCodePuller(filepath.Join(opts.Worker_dir, "lambda_code"), opts.Registry)
	if err != nil {
		return nil, err
	}
	log.Printf("Initialized registry manager (took %v)", time.Since(t))

	t = time.Now()
	pm, err := pip.InitInstallManager(opts)
	if err != nil {
		return nil, err
	}
	log.Printf("Initialized installation manager (took %v)", time.Since(t))

	t = time.Now()
	sf, err := sb.InitHandlerContainerFactory(opts)
	if err != nil {
		return nil, err
	}
	log.Printf("Initialized handler container factory (took %v)", time.Since(t))

	t = time.Now()
	cm, err := cache.InitCacheManager(opts)
	if err != nil {
		return nil, err
	}
	log.Printf("Initialized cache manager (took %v)", time.Since(t))

	var hhits int64 = 0
	var ihits int64 = 0
	var misses int64 = 0
	mgr = &LambdaMgr{
		lfuncMap:   make(map[string]*LambdaFunc),
		codePuller:     cp,
		pipMgr:     pm,
		sbFactory:  sf,
		cacheMgr:   cm,
		config:     opts,
		workerDir:  opts.Worker_dir,
		maxRunners: opts.Max_runners,
		hhits:      &hhits,
		ihits:      &ihits,
		misses:     &misses,
	}

	mgr.lru = NewLambdaInstanceLRU(mgr, opts.Handler_cache_size) //kb

	return mgr, nil
}

// Returns an existing instance (if there is one), or creates a new one
func (mgr *LambdaMgr) Get(name string) (linst *LambdaInstance, err error) {
	mgr.mutex.Lock()

	lfunc := mgr.lfuncMap[name]

	if lfunc == nil {
		workingDir := filepath.Join(mgr.workerDir, "handlers", name)
		mgr.lfuncMap[name] = &LambdaFunc{
			name:       name,
			lmgr:       mgr,
			instances:  list.New(),
			listEl:     make(map[*LambdaInstance]*list.Element),
			workingDir: workingDir,
			imports:    []string{},
			installs:   []string{},
		}

		lfunc = mgr.lfuncMap[name]
	}

	// find or create instance
	lfunc.mutex.Lock()
	if lfunc.instances.Front() == nil {
		linst = &LambdaInstance{
			name:    name,
			lfunc:   lfunc,
			runners: 1,
		}
	} else {
		listEl := lfunc.instances.Front()
		linst = listEl.Value.(*LambdaInstance)

		// remove from lru if necessary
		linst.mutex.Lock()
		if linst.runners == 0 {
			mgr.lru.Remove(linst)
		}

		linst.runners += 1

		if linst.lfunc.lmgr.maxRunners != 0 && linst.runners == linst.lfunc.lmgr.maxRunners {
			lfunc.instances.Remove(listEl)
			delete(lfunc.listEl, linst)
		}
		linst.mutex.Unlock()
	}
	// not perfect, but removal from the LRU needs to be atomic
	// with respect to the LRU and the LambdaMgr
	mgr.mutex.Unlock()

	// get code if needed
	if lfunc.lastPull == nil {
		codeDir, err := mgr.codePuller.Pull(lfunc.name)
		if err != nil {
			return nil, err
		}

		imports, installs, err := parsePkgFile(codeDir)
		if err != nil {
			return nil, err
		}
		
		now := time.Now()
		lfunc.lastPull = &now
		lfunc.codeDir = codeDir
		lfunc.imports = imports
		lfunc.installs = installs
	}
	lfunc.mutex.Unlock()

	return linst, nil
}

// Dump prints the name and state of the instances currently in the LambdaMgr.
func (mgr *LambdaMgr) Dump() {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	log.Printf("LAMBDA INSTANCES:\n")
	for name, lfunc := range mgr.lfuncMap {
		lfunc.mutex.Lock()
		log.Printf(" %v: %d", name, lfunc.maxInstances)
		for e := lfunc.instances.Front(); e != nil; e = e.Next() {
			linst := e.Value.(*LambdaInstance)
			state, _ := linst.sandbox.State()
			log.Printf(" > %v: %v\n", linst.id, state.String())
		}
		lfunc.mutex.Unlock()
	}
}

func (mgr *LambdaMgr) Cleanup() {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	for _, lfunc := range mgr.lfuncMap {
		for e := lfunc.instances.Front(); e != nil; e = e.Next() {
			e.Value.(*LambdaInstance).nuke()
		}
	}

	mgr.sbFactory.Cleanup()

	if mgr.cacheMgr != nil {
		mgr.cacheMgr.Cleanup()
	}
}

// must be called with instance lock
func (lfunc *LambdaFunc) AddInstance(linst *LambdaInstance) {
	mgr := lfunc.lmgr

	// if we finish first
	// no deadlock can occur here despite taking the locks in the
	// opposite order because hm -> h in Get has no reference
	// in the instance list
	if mgr.maxRunners != 0 && linst.runners == mgr.maxRunners-1 {
		lfunc.mutex.Lock()
		lfunc.listEl[linst] = lfunc.instances.PushFront(linst)
		lfunc.maxInstances = max(lfunc.maxInstances, lfunc.instances.Len())
		lfunc.mutex.Unlock()
	}
}

func (lfunc *LambdaFunc) TryRemoveInstance(linst *LambdaInstance) error {
	lfunc.mutex.Lock()
	defer lfunc.mutex.Unlock()
	linst.mutex.Lock()
	defer linst.mutex.Unlock()

	// someone has come in and has started running
	if linst.runners > 0 {
		return errors.New("concurrent runner entered system")
	}

	// remove reference to instance in LambdaMgr
	// this ensures h is the last reference to the Instance
	if listEl := lfunc.listEl[linst]; listEl != nil {
		lfunc.instances.Remove(listEl)
		delete(lfunc.listEl, linst)
	}

	return nil
}

// RunStart runs the lambda handled by this Instance. It checks if the code has
// been pulled, sandbox been created, and sandbox been started. The channel of
// the sandbox of this lambda is returned.
func (linst *LambdaInstance) RunStart() (ch *sb.Channel, err error) {
	linst.mutex.Lock()
	defer linst.mutex.Unlock()

	lfunc := linst.lfunc
	mgr := linst.lfunc.lmgr

	// create sandbox if needed
	if linst.sandbox == nil {
		hit := false

		// TODO: do this in the background
		err = mgr.pipMgr.Install(lfunc.installs)
		if err != nil {
			return nil, err
		}

		sandbox, err := mgr.sbFactory.Create(lfunc.codeDir, lfunc.workingDir)
		if err != nil {
			return nil, err
		}

		linst.sandbox = sandbox
		linst.id = linst.sandbox.ID()
		linst.hostDir = linst.sandbox.HostDir()

		if sbState, err := linst.sandbox.State(); err != nil {
			return nil, err
		} else if sbState == state.Stopped {
			if err := linst.sandbox.Start(); err != nil {
				return nil, err
			}
		} else if sbState == state.Paused {
			if err := linst.sandbox.Unpause(); err != nil {
				return nil, err
			}
		}

		if mgr.cacheMgr == nil {
			if err := linst.sandbox.RunServer(); err != nil {
				return nil, err
			}
		} else {
			if linst.fs, hit, err = mgr.cacheMgr.Provision(sandbox, lfunc.imports); err != nil {
				return nil, err
			}

			if hit {
				atomic.AddInt64(mgr.ihits, 1)
			} else {
				atomic.AddInt64(mgr.misses, 1)
			}
		}

		// use StdoutPipe of olcontainer to sync with lambda server
		ready := make(chan bool, 1)
		defer close(ready)
		go func() {
			pipeDir := filepath.Join(linst.hostDir, "server_pipe")
			pipe, err := os.OpenFile(pipeDir, os.O_RDWR, 0777)
			if err != nil {
				log.Printf("Cannot open pipe: %v\n", err)
				return
			}
			defer pipe.Close()

			// wait for "ready"
			buf := make([]byte, 5)
			_, err = pipe.Read(buf)
			if err != nil {
				log.Printf("Cannot read from stdout of sandbox :: %v\n", err)
			} else if string(buf) != "ready" {
				log.Printf("Expect to see `ready` but got %s\n", string(buf))
			}
			ready <- true
		}()

		// wait up to 20s for server to initialize
		start := time.Now()
		timeout := time.NewTimer(20 * time.Second)
		defer timeout.Stop()

		select {
		case <-ready:
			if config.Timing {
				log.Printf("wait for server took %v\n", time.Since(start))
			}
		case <-timeout.C:
			return nil, fmt.Errorf("instance server failed to initialize after 20s")
		}

		// we are up so we can add ourselves for reuse
		if mgr.maxRunners == 0 || linst.runners < mgr.maxRunners {
			lfunc.mutex.Lock()
			lfunc.listEl[linst] = lfunc.instances.PushFront(linst)
			lfunc.maxInstances = max(lfunc.maxInstances, lfunc.instances.Len())
			lfunc.mutex.Unlock()
		}

	} else if sbState, _ := linst.sandbox.State(); sbState == state.Paused {
		// unpause if paused
		atomic.AddInt64(mgr.hhits, 1)
		if err := linst.sandbox.Unpause(); err != nil {
			return nil, err
		}
	} else {
		atomic.AddInt64(mgr.hhits, 1)
	}

	log.Printf("handler hits: %v, import hits: %v, misses: %v", *mgr.hhits, *mgr.ihits, *mgr.misses)
	return linst.sandbox.Channel()
}

// RunFinish notifies that a request to run the lambda has completed. If no
// request is being run in its sandbox, sandbox will be paused and the instance
// be added to the InstanceLRU.
func (linst *LambdaInstance) RunFinish() {
	linst.mutex.Lock()

	lfunc := linst.lfunc
	mgr := linst.lfunc.lmgr

	linst.runners -= 1

	// are we the last?
	if linst.runners == 0 {
		if err := linst.sandbox.Pause(); err != nil {
			// TODO(tyler): better way to handle this?  If
			// we can't pause, the instance gets to keep
			// running for free...
			log.Printf("Could not pause %v: %v!  Error: %v\n", linst.name, linst.id, err)
		}

		if lambdaInstanceUsage(linst) > mgr.lru.soft_limit {
			linst.mutex.Unlock()

			// we were potentially the last runner
			// try to remove us from the instance manager
			if err := lfunc.TryRemoveInstance(linst); err == nil {
				// we were the last one so... bye
				go linst.nuke()
			}
			return
		}

		lfunc.AddInstance(linst)
		mgr.lru.Add(linst)
	} else {
		lfunc.AddInstance(linst)
	}

	linst.mutex.Unlock()
}

func (linst *LambdaInstance) nuke() {
	if err := linst.sandbox.Unpause(); err != nil {
		log.Printf("failed to unpause sandbox :: %v", err.Error())
	}
	if err := linst.sandbox.Stop(); err != nil {
		log.Printf("failed to stop sandbox :: %v", err.Error())
	}
	if err := linst.sandbox.Remove(); err != nil {
		log.Printf("failed to remove sandbox :: %v", err.Error())
	}
}

// Sandbox returns the sandbox of this Instance.
func (linst *LambdaInstance) Sandbox() sb.Sandbox {
	return linst.sandbox
}

func max(i1, i2 int) int {
	if i1 < i2 {
		return i2
	}
	return i1
}

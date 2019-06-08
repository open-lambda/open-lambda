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
	"time"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/handler/state"
	"github.com/open-lambda/open-lambda/ol/pip-manager"

	sb "github.com/open-lambda/open-lambda/ol/sandbox"
)

// Organizes all lambda functions (the code) and lambda instances (that serve events)
type LambdaMgr struct {
	mutex      sync.Mutex
	lfuncMap   map[string]*LambdaFunc
	codePuller *CodePuller
	pipMgr     pip.InstallManager
	sbFactory  sb.ContainerFactory
	lru        *LambdaInstanceLRU
	workerDir  string
	maxRunners int
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
	sandbox sb.Sandbox
	runners int
	usage   int
}

func NewLambdaMgr() (mgr *LambdaMgr, err error) {
	var t time.Time

	// start with a fresh worker directory
	log.Printf("Cleanup old dirs")
	if err := os.RemoveAll(config.Conf.Worker_dir); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(config.Conf.Worker_dir, 0700); err != nil {
		return nil, err
	}

	// init code puller, pip manager, handler cache, and init cache
	log.Printf("Create CodePuller")
	t = time.Now()
	cp, err := NewCodePuller(filepath.Join(config.Conf.Worker_dir, "lambda_code"), config.Conf.Registry)
	if err != nil {
		return nil, err
	}
	log.Printf("Initialized CodePuller (took %v)", time.Since(t))

	log.Printf("Create InstallManager")
	t = time.Now()
	pm, err := pip.InitInstallManager()
	if err != nil {
		return nil, err
	}
	log.Printf("Create InstallManager (took %v)", time.Since(t))

	log.Printf("Create ContainerFactory")
	t = time.Now()
	sf, err := sb.InitHandlerContainerFactory()
	if err != nil {
		return nil, err
	}
	log.Printf("Initialized handler container factory (took %v)", time.Since(t))

	t = time.Now()

	mgr = &LambdaMgr{
		lfuncMap:   make(map[string]*LambdaFunc),
		codePuller: cp,
		pipMgr:     pm,
		sbFactory:  sf,
		workerDir:  config.Conf.Worker_dir,
		maxRunners: config.Conf.Max_runners,
	}

	mgr.lru = NewLambdaInstanceLRU(mgr, config.Conf.Handler_cache_mb)

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

		if mgr.maxRunners != 0 && linst.runners == mgr.maxRunners {
			lfunc.instances.Remove(listEl)
			delete(lfunc.listEl, linst)
		}
		linst.mutex.Unlock()
	}
	// not perfect, but removal from the LRU needs to be atomic
	// with respect to the LRU and the LambdaMgr
	mgr.mutex.Unlock()

	// get code if needed
	now := time.Now()
	cache_ns := int64(config.Conf.Registry_cache_ms) * 1000000
	if lfunc.lastPull == nil || int64(now.Sub(*lfunc.lastPull)) > cache_ns {
		codeDir, err := mgr.codePuller.Pull(lfunc.name)
		if err != nil {
			return nil, err
		}

		imports, installs, err := parsePkgFile(codeDir)
		if err != nil {
			return nil, err
		}

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

	log.Printf("Cleanup Lambdas:\n")
	for _, lfunc := range mgr.lfuncMap {
		log.Printf("  Function: %s\n", lfunc.name)
		for e := lfunc.instances.Front(); e != nil; e = e.Next() {
			log.Printf("    Instance: %s\n", e.Value.(*LambdaInstance).id)
			e.Value.(*LambdaInstance).sandbox.Destroy()
		}
	}

	log.Printf("Cleanup Container Factory\n")
	mgr.sbFactory.Cleanup()

	log.Printf("Finished Lambda Cleanup\n")
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
		err = mgr.pipMgr.Install(lfunc.installs)
		if err != nil {
			return nil, err
		}

		sandbox, err := mgr.sbFactory.Create(lfunc.codeDir, lfunc.workingDir, lfunc.imports)
		if err != nil {
			return nil, err
		}

		linst.sandbox = sandbox
		linst.id = linst.sandbox.ID()

		// use StdoutPipe of olcontainer to sync with lambda server
		ready := make(chan bool, 1)
		defer close(ready)
		go func() {
			pipeDir := filepath.Join(linst.sandbox.HostDir(), "server_pipe")
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
			if config.Conf.Timing {
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
		if err := linst.sandbox.Unpause(); err != nil {
			return nil, err
		}
	}

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

		lfunc.AddInstance(linst)
		mgr.lru.Add(linst)
	} else {
		lfunc.AddInstance(linst)
	}

	linst.mutex.Unlock()
}

func max(i1, i2 int) int {
	if i1 < i2 {
		return i2
	}
	return i1
}

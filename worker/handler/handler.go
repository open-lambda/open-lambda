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
	"github.com/open-lambda/open-lambda/worker/registry"

	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

// HandlerSet represents a collection of Handlers of a worker server. It
// manages the Handler by HandlerLRU.
type HandlerManagerSet struct {
	mutex      sync.Mutex
	hmMap      map[string]*HandlerManager
	regMgr     registry.RegistryManager
	pipMgr     pip.InstallManager
	sbFactory  sb.ContainerFactory
	cacheMgr   *cache.CacheManager
	config     *config.Config
	lru        *HandlerLRU
	workerDir  string
	maxRunners int
	hhits      *int64
	ihits      *int64
	misses     *int64
}

type HandlerManager struct {
	name        string
	mutex       sync.Mutex
	hms         *HandlerManagerSet
	handlers    *list.List
	hElements   map[*Handler]*list.Element
	workingDir  string
	maxHandlers int
	lastPull    *time.Time
	code        []byte
	codeDir     string
	imports     []string
	installs    []string
}

// Handler handles requests to run a lambda on a worker server. It handles
// concurrency and communicates with the sandbox manager to change the
// state of the container that servers the lambda.
type Handler struct {
	name    string
	id      string
	mutex   sync.Mutex
	hm      *HandlerManager
	sandbox sb.Container
	fs      *cache.ForkServer
	hostDir string
	runners int
	usage   int
}

// NewHandlerSet creates an empty HandlerSet
func NewHandlerManagerSet(opts *config.Config) (hms *HandlerManagerSet, err error) {
	var t time.Time

	t = time.Now()
	rm, err := registry.InitRegistryManager(opts)
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
	hms = &HandlerManagerSet{
		hmMap:      make(map[string]*HandlerManager),
		regMgr:     rm,
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

	hms.lru = NewHandlerLRU(hms, opts.Handler_cache_size) //kb

	return hms, nil
}

// Get always returns a Handler, creating one if necessarily.
func (hms *HandlerManagerSet) Get(name string) (h *Handler, err error) {
	hms.mutex.Lock()

	hm := hms.hmMap[name]

	if hm == nil {
		workingDir := filepath.Join(hms.workerDir, "handlers", name)
		hms.hmMap[name] = &HandlerManager{
			name:       name,
			hms:        hms,
			handlers:   list.New(),
			hElements:  make(map[*Handler]*list.Element),
			workingDir: workingDir,
			imports:    []string{},
			installs:   []string{},
		}

		hm = hms.hmMap[name]
	}

	// find or create handler
	hm.mutex.Lock()
	if hm.handlers.Front() == nil {
		h = &Handler{
			name:    name,
			hm:      hm,
			runners: 1,
		}
	} else {
		hEle := hm.handlers.Front()
		h = hEle.Value.(*Handler)

		// remove from lru if necessary
		h.mutex.Lock()
		if h.runners == 0 {
			hms.lru.Remove(h)
		}

		h.runners += 1

		if h.hm.hms.maxRunners != 0 && h.runners == h.hm.hms.maxRunners {
			hm.handlers.Remove(hEle)
			delete(hm.hElements, h)
		}
		h.mutex.Unlock()
	}
	// not perfect, but removal from the LRU needs to be atomic
	// with respect to the LRU and the HandlerManager
	hms.mutex.Unlock()

	// get code if needed
	if hm.lastPull == nil {
		codeDir, imports, installs, err := hms.regMgr.Pull(hm.name)
		if err != nil {
			return nil, err
		}

		now := time.Now()
		hm.lastPull = &now
		hm.codeDir = codeDir
		hm.imports = imports
		hm.installs = installs
	}
	hm.mutex.Unlock()

	return h, nil
}

// Dump prints the name and state of the Handlers currently in the HandlerSet.
func (hms *HandlerManagerSet) Dump() {
	hms.mutex.Lock()
	defer hms.mutex.Unlock()

	log.Printf("HANDLERS:\n")
	for name, hm := range hms.hmMap {
		hm.mutex.Lock()
		log.Printf(" %v: %d", name, hm.maxHandlers)
		for e := hm.handlers.Front(); e != nil; e = e.Next() {
			h := e.Value.(*Handler)
			state, _ := h.sandbox.State()
			log.Printf(" > %v: %v\n", h.id, state.String())
		}
		hm.mutex.Unlock()
	}
}

func (hms *HandlerManagerSet) Cleanup() {
	hms.mutex.Lock()
	defer hms.mutex.Unlock()

	for _, hm := range hms.hmMap {
		for e := hm.handlers.Front(); e != nil; e = e.Next() {
			e.Value.(*Handler).nuke()
		}
	}

	hms.sbFactory.Cleanup()

	if hms.cacheMgr != nil {
		hms.cacheMgr.Cleanup()
	}
}

// must be called with handler lock
func (hm *HandlerManager) AddHandler(h *Handler) {
	hms := hm.hms

	// if we finish first
	// no deadlock can occur here despite taking the locks in the
	// opposite order because hm -> h in Get has no reference
	// in the handler list
	if hms.maxRunners != 0 && h.runners == hms.maxRunners-1 {
		hm.mutex.Lock()
		hm.hElements[h] = hm.handlers.PushFront(h)
		hm.maxHandlers = max(hm.maxHandlers, hm.handlers.Len())
		hm.mutex.Unlock()
	}
}

func (hm *HandlerManager) TryRemoveHandler(h *Handler) error {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// someone has come in and has started running
	if h.runners > 0 {
		return errors.New("concurrent runner entered system")
	}

	// remove reference to handler in HandlerManager
	// this ensures h is the last reference to the Handler
	if hEle := hm.hElements[h]; hEle != nil {
		hm.handlers.Remove(hEle)
		delete(hm.hElements, h)
	}

	return nil
}

// RunStart runs the lambda handled by this Handler. It checks if the code has
// been pulled, sandbox been created, and sandbox been started. The channel of
// the sandbox of this lambda is returned.
func (h *Handler) RunStart() (ch *sb.Channel, err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	hm := h.hm
	hms := h.hm.hms

	// create sandbox if needed
	if h.sandbox == nil {
		hit := false

		// TODO: do this in the background
		err = hms.pipMgr.Install(hm.installs)
		if err != nil {
			return nil, err
		}

		sandbox, err := hms.sbFactory.Create(hm.codeDir, hm.workingDir)
		if err != nil {
			return nil, err
		}

		h.sandbox = sandbox
		h.id = h.sandbox.ID()
		h.hostDir = h.sandbox.HostDir()

		if sbState, err := h.sandbox.State(); err != nil {
			return nil, err
		} else if sbState == state.Stopped {
			if err := h.sandbox.Start(); err != nil {
				return nil, err
			}
		} else if sbState == state.Paused {
			if err := h.sandbox.Unpause(); err != nil {
				return nil, err
			}
		}

		if hms.cacheMgr == nil {
			if err := h.sandbox.RunServer(); err != nil {
				return nil, err
			}
		} else {
			if h.fs, hit, err = hms.cacheMgr.Provision(sandbox, hm.imports); err != nil {
				return nil, err
			}

			if hit {
				atomic.AddInt64(hms.ihits, 1)
			} else {
				atomic.AddInt64(hms.misses, 1)
			}
		}

		// use StdoutPipe of olcontainer to sync with lambda server
		ready := make(chan bool, 1)
		defer close(ready)
		go func() {
			pipeDir := filepath.Join(h.hostDir, "server_pipe")
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
			return nil, fmt.Errorf("handler server failed to initialize after 20s")
		}

		// we are up so we can add ourselves for reuse
		if hms.maxRunners == 0 || h.runners < hms.maxRunners {
			hm.mutex.Lock()
			hm.hElements[h] = hm.handlers.PushFront(h)
			hm.maxHandlers = max(hm.maxHandlers, hm.handlers.Len())
			hm.mutex.Unlock()
		}

	} else if sbState, _ := h.sandbox.State(); sbState == state.Paused {
		// unpause if paused
		atomic.AddInt64(hms.hhits, 1)
		if err := h.sandbox.Unpause(); err != nil {
			return nil, err
		}
	} else {
		atomic.AddInt64(hms.hhits, 1)
	}

	log.Printf("handler hits: %v, import hits: %v, misses: %v", *hms.hhits, *hms.ihits, *hms.misses)
	return h.sandbox.Channel()
}

// RunFinish notifies that a request to run the lambda has completed. If no
// request is being run in its sandbox, sandbox will be paused and the handler
// be added to the HandlerLRU.
func (h *Handler) RunFinish() {
	h.mutex.Lock()

	hm := h.hm
	hms := h.hm.hms

	h.runners -= 1

	// are we the last?
	if h.runners == 0 {
		if err := h.sandbox.Pause(); err != nil {
			// TODO(tyler): better way to handle this?  If
			// we can't pause, the handler gets to keep
			// running for free...
			log.Printf("Could not pause %v: %v!  Error: %v\n", h.name, h.id, err)
		}

		if handlerUsage(h) > hms.lru.soft_limit {
			h.mutex.Unlock()

			// we were potentially the last runner
			// try to remove us from the handler manager
			if err := hm.TryRemoveHandler(h); err == nil {
				// we were the last one so... bye
				go h.nuke()
			}
			return
		}

		hm.AddHandler(h)
		hms.lru.Add(h)
	} else {
		hm.AddHandler(h)
	}

	h.mutex.Unlock()
}

func (h *Handler) nuke() {
	if err := h.sandbox.Unpause(); err != nil {
		log.Printf("failed to unpause sandbox :: %v", err.Error())
	}
	if err := h.sandbox.Stop(); err != nil {
		log.Printf("failed to stop sandbox :: %v", err.Error())
	}
	if err := h.sandbox.Remove(); err != nil {
		log.Printf("failed to remove sandbox :: %v", err.Error())
	}
}

// Sandbox returns the sandbox of this Handler.
func (h *Handler) Sandbox() sb.Sandbox {
	return h.sandbox
}

func max(i1, i2 int) int {
	if i1 < i2 {
		return i2
	}
	return i1
}

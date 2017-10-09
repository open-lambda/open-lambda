// handler package implements a library for handling run lambda requests from
// the worker server.
package handler

import (
	"container/list"
	"fmt"
	"log"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler/state"
	"github.com/open-lambda/open-lambda/worker/import-cache"
	"github.com/open-lambda/open-lambda/worker/registry"

	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

// HandlerSet represents a collection of Handlers of a worker server. It
// manages the Handler by HandlerLRU.
type HandlerManagerSet struct {
	mutex      sync.Mutex
	hmMap      map[string]*HandlerManager
	regMgr     registry.RegistryManager
	sbFactory  sb.SandboxFactory
	cacheMgr   *cache.CacheManager
	config     *config.Config
	lru        *HandlerLRU
	workerDir  string
	indexHost  string
	indexPort  string
	maxRunners int
	hhits      *int64
	ihits      *int64
	misses     *int64
}

type HandlerManager struct {
	mutex      sync.Mutex
	hms        *HandlerManagerSet
	handlers   *list.List
	hElements  map[*Handler]*list.Element
	workingDir string
	lastPull   *time.Time
	code       []byte
	codeDir    string
	pkgs       []string
}

// Handler handles requests to run a lambda on a worker server. It handles
// concurrency and communicates with the sandbox manager to change the
// state of the container that servers the lambda.
type Handler struct {
	name    string
	id      string
	mutex   sync.Mutex
	hm      *HandlerManager
	sandbox sb.Sandbox
	fs      *cache.ForkServer
	hostDir string
	runners int
	usage   int
}

// NewHandlerSet creates an empty HandlerSet
func NewHandlerManagerSet(opts *config.Config) (hms *HandlerManagerSet, err error) {
	rm, err := registry.InitRegistryManager(opts)
	if err != nil {
		return nil, err
	}

	sf, err := sb.InitSandboxFactory(opts)
	if err != nil {
		return nil, err
	}

	cm, err := cache.InitCacheManager(opts)
	if err != nil {
		return nil, err
	}

	var hhits int64 = 0
	var ihits int64 = 0
	var misses int64 = 0
	hms = &HandlerManagerSet{
		hmMap:      make(map[string]*HandlerManager),
		regMgr:     rm,
		sbFactory:  sf,
		cacheMgr:   cm,
		config:     opts,
		workerDir:  opts.Worker_dir,
		indexHost:  opts.Index_host,
		indexPort:  opts.Index_port,
		maxRunners: opts.Max_runners,
		hhits:      &hhits,
		ihits:      &ihits,
		misses:     &misses,
	}

	hms.lru = NewHandlerLRU(hms, opts.Handler_cache_size) //kb

	/*
		if cm != nil {
			go handlerSet.killOrphans()
		}
	*/

	return hms, nil
}

// Get always returns a Handler, creating one if necessarily.
func (hms *HandlerManagerSet) Get(name string) (h *Handler, err error) {
	hms.mutex.Lock()

	hm := hms.hmMap[name]

	if hm == nil {
		workingDir := path.Join(hms.workerDir, "handlers", name)
		hms.hmMap[name] = &HandlerManager{
			hms:        hms,
			handlers:   list.New(),
			hElements:  make(map[*Handler]*list.Element),
			workingDir: workingDir,
			pkgs:       []string{},
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
		codeDir, pkgs, err := hms.regMgr.Pull(h.name)
		if err != nil {
			return nil, err
		}

		now := time.Now()
		hm.lastPull = &now
		hm.codeDir = codeDir
		hm.pkgs = pkgs
	}
	hm.mutex.Unlock()

	return h, nil
}

/*
func (h *HandlerSet) killOrphans() {
	var toDelete string
	for {
		if toDelete != "" {
			h.mutex.Lock()
			handler := h.handlers[toDelete]
			delete(h.handlers, toDelete)
			h.mutex.Unlock()
			go handler.nuke()
		}
		toDelete = ""
		for _, handler := range h.handlers {
			handler.mutex.Lock()
			if handler.fs != nil && handler.fs.Dead {
				toDelete = handler.name
			}
			time.Sleep(50 * time.Microsecond)
			handler.mutex.Unlock()
		}
	}
}
*/

// Dump prints the name and state of the Handlers currently in the HandlerSet.
func (hms *HandlerManagerSet) Dump() {
	hms.mutex.Lock()
	defer hms.mutex.Unlock()

	log.Printf("HANDLERS:\n")
	for name, hm := range hms.hmMap {
		hm.mutex.Lock()
		for e := hm.handlers.Front(); e != nil; e = e.Next() {
			state, _ := e.Value.(*Handler).sandbox.State()
			log.Printf("> %v: %v\n", name, state.String())
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

// RunStart runs the lambda handled by this Handler. It checks if the code has
// been pulled, sandbox been created, and sandbox been started. The channel of
// the sandbox of this lambda is returned.
func (h *Handler) RunStart() (ch *sb.SandboxChannel, err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	hm := h.hm
	hms := h.hm.hms

	// create sandbox if needed
	if h.sandbox == nil {
		sandbox, err := hms.sbFactory.Create(hm.codeDir, hm.workingDir, hms.indexHost, hms.indexPort)
		if err != nil {
			return nil, err
		}

		h.sandbox = sandbox
		h.id = h.sandbox.ID()
		h.hostDir = path.Join(hm.workingDir, h.id)
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

		hit := false
		if hms.cacheMgr == nil || hms.cacheMgr.Full() {
			err := h.sandbox.RunServer()
			if err != nil {
				return nil, err
			}
		} else {
			containerSB, ok := h.sandbox.(sb.ContainerSandbox)
			if !ok {
				return nil, fmt.Errorf("forkenter only supported with ContainerSandbox")
			}
			if h.fs, hit, err = hms.cacheMgr.Provision(containerSB, h.hostDir, hm.pkgs); err != nil {
				return nil, err
			}

		}

		if hit {
			atomic.AddInt64(hms.ihits, 1)
		} else {
			atomic.AddInt64(hms.misses, 1)
		}

		sockPath := fmt.Sprintf("%s/ol.sock", h.hostDir)

		// wait up to 20s for server to initialize
		start := time.Now()
		for ok := true; ok; ok = os.IsNotExist(err) {
			_, err = os.Stat(sockPath)
			if hms.config.Sandbox == "olcontainer" && (hms.cacheMgr == nil || hms.cacheMgr.Full()) {
				time.Sleep(10 * time.Microsecond)
				if err := h.sandbox.RunServer(); err != nil {
					return nil, err
				}
			}
			if time.Since(start).Seconds() > 20 {
				return nil, fmt.Errorf("handler server failed to initialize after 20s")
			}
			time.Sleep(50 * time.Microsecond)
		}

		// we are up so we can add ourselves for reuse
		if hms.maxRunners == 0 || h.runners < hms.maxRunners {
			hm.mutex.Lock()
			hm.hElements[h] = hm.handlers.PushFront(h)
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
	defer h.mutex.Unlock()

	hm := h.hm
	hms := h.hm.hms

	// if we finish first
	// no deadlock can occur here despite taking the locks in the
	// opposite order because hm -> h in Get has no reference
	// in the handler list
	if hms.maxRunners != 0 && h.runners == hms.maxRunners {
		hm.mutex.Lock()
		hm.hElements[h] = hm.handlers.PushFront(h)
		hm.mutex.Unlock()
	}

	h.runners -= 1

	// are we the last?
	if h.runners == 0 {
		if err := h.sandbox.Pause(); err != nil {
			// TODO(tyler): better way to handle this?  If
			// we can't pause, the handler gets to keep
			// running for free...
			log.Printf("Could not pause %v!  Error: %v\n", h.name, err)
		}

		hms.lru.Add(h)
	}
}

func (h *Handler) nuke() {
	h.sandbox.Unpause()
	h.sandbox.Stop()
	h.sandbox.Remove()
}

// Sandbox returns the sandbox of this Handler.
func (h *Handler) Sandbox() sb.Sandbox {
	return h.sandbox
}

// handler package implements a library for handling run lambda requests from
// the worker server.
package handler

import (
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
type HandlerSet struct {
	mutex       sync.Mutex
	handlers    map[string]*Handler
	pullMutexes map[string]*sync.Mutex
	regMgr      registry.RegistryManager
	sbFactory   sb.SandboxFactory
	cacheMgr    *cache.CacheManager
	config      *config.Config
	lru         *HandlerLRU
	workerDir   string
	indexHost   string
	indexPort   string
	hhits       *int64
	ihits       *int64
	misses      *int64
}

// Handler handles requests to run a lambda on a worker server. It handles
// concurrency and communicates with the sandbox manager to change the
// state of the container that servers the lambda.
type Handler struct {
	name     string
	id       string
	mutex    sync.Mutex
	hset     *HandlerSet
	sandbox  sb.Sandbox
	lastPull *time.Time
	runners  int
	code     []byte
	codeDir  string
	pkgs     []string
	hostDir  string
	fs       *cache.ForkServer
	usage    int
}

// NewHandlerSet creates an empty HandlerSet
func NewHandlerSet(opts *config.Config) (handlerSet *HandlerSet, err error) {
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
	handlers := make(map[string]*Handler)
	handlerSet = &HandlerSet{
		handlers:    handlers,
		pullMutexes: make(map[string]*sync.Mutex),
		regMgr:      rm,
		sbFactory:   sf,
		cacheMgr:    cm,
		workerDir:   opts.Worker_dir,
		indexHost:   opts.Index_host,
		indexPort:   opts.Index_port,
		hhits:       &hhits,
		ihits:       &ihits,
		misses:      &misses,
	}

	handlerSet.lru = NewHandlerLRU(handlerSet, opts.Handler_cache_size) //kb

	/*
		if cm != nil {
			go handlerSet.killOrphans()
		}
	*/

	return handlerSet, nil
}

// Get always returns a Handler, creating one if necessarily.
func (h *HandlerSet) Get(name string) *Handler {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	handler := h.handlers[name]
	if handler == nil {
		hostDir := path.Join(h.workerDir, "handlers", name)
		handler = &Handler{
			name:    name,
			hset:    h,
			runners: 0,
			pkgs:    []string{},
			hostDir: hostDir,
		}
		h.handlers[name] = handler
		h.pullMutexes[name] = &sync.Mutex{}
	}

	return handler
}

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

// Dump prints the name and state of the Handlers currently in the HandlerSet.
func (h *HandlerSet) Dump() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	log.Printf("HANDLERS:\n")
	for k, v := range h.handlers {
		state, _ := v.sandbox.State()
		log.Printf("> %v: %v\n", k, state.String())
	}
}

func (h *HandlerSet) Cleanup() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for _, handler := range h.handlers {
		handler.nuke()
	}
	h.sbFactory.Cleanup()
	if h.cacheMgr != nil {
		h.cacheMgr.Cleanup()
	}
}

// RunStart runs the lambda handled by this Handler. It checks if the code has
// been pulled, sandbox been created, and sandbox been started. The channel of
// the sandbox of this lambda is returned.
func (h *Handler) RunStart() (ch *sb.SandboxChannel, err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// get code if needed
	if h.lastPull == nil {
		// get pull mutex
		h.hset.mutex.Lock()
		pullMutex := h.hset.pullMutexes[h.name]
		h.hset.mutex.Unlock()

		pullMutex.Lock()

		codeDir, pkgs, err := h.hset.regMgr.Pull(h.name)
		if err != nil {
			return nil, err
		}

		now := time.Now()
		h.lastPull = &now
		h.codeDir = codeDir
		h.pkgs = pkgs

		pullMutex.Unlock()
	}

	// create sandbox if needed
	if h.sandbox == nil {
		sandbox, err := h.hset.sbFactory.Create(h.codeDir, h.hostDir, h.hset.indexHost, h.hset.indexPort)
		if err != nil {
			return nil, err
		}

		h.sandbox = sandbox
		h.id = h.sandbox.ID()
		h.hostDir = path.Join(h.hostDir, h.id)
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
		if h.hset.cacheMgr == nil || h.hset.cacheMgr.Full() {
			err := h.sandbox.RunServer()
			if err != nil {
				return nil, err
			}
		} else {
			containerSB, ok := h.sandbox.(sb.ContainerSandbox)
			if !ok {
				return nil, fmt.Errorf("forkenter only supported with ContainerSandbox")
			}
			if h.fs, hit, err = h.hset.cacheMgr.Provision(containerSB, h.hostDir, h.pkgs); err != nil {
				return nil, err
			}

		}

		if hit {
			atomic.AddInt64(h.hset.ihits, 1)
		} else {
			atomic.AddInt64(h.hset.misses, 1)
		}

		sockPath := fmt.Sprintf("%s/ol.sock", h.hostDir)

		// wait up to 20s for server to initialize
		start := time.Now()
		for ok := true; ok; ok = os.IsNotExist(err) {
			_, err = os.Stat(sockPath)
			if time.Since(start).Seconds() > 20 {
				return nil, fmt.Errorf("handler server failed to initialize after 20s")
			}
			time.Sleep(50 * time.Microsecond)
		}

	} else if sbState, _ := h.sandbox.State(); sbState == state.Paused {
		// unpause if paused
		atomic.AddInt64(h.hset.hhits, 1)
		if err := h.sandbox.Unpause(); err != nil {
			return nil, err
		}
		h.hset.lru.Remove(h)
	} else {
		atomic.AddInt64(h.hset.hhits, 1)
	}

	h.runners += 1

	log.Printf("handler hits: %v, import hits: %v, misses: %v", *h.hset.hhits, *h.hset.ihits, *h.hset.misses)
	return h.sandbox.Channel()
}

// RunFinish notifies that a request to run the lambda has completed. If no
// request is being run in its sandbox, sandbox will be paused and the handler
// be added to the HandlerLRU.
func (h *Handler) RunFinish() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.runners -= 1

	// are we the last?
	if h.runners == 0 {
		if err := h.sandbox.Pause(); err != nil {
			// TODO(tyler): better way to handle this?  If
			// we can't pause, the handler gets to keep
			// running for free...
			log.Printf("Could not pause %v!  Error: %v\n", h.name, err)
		}
		h.hset.lru.Add(h)
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

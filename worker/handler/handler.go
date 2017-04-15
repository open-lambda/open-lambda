// handler package implements a library for handling run lambda requests from
// the worker server.
package handler

import (
	"errors"
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler/state"
	"github.com/open-lambda/open-lambda/worker/registry"

	pmanager "github.com/open-lambda/open-lambda/worker/pool-manager"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

// HandlerSetOpts wraps parameters necessary to create a HandlerSet.
type HandlerSetOpts struct {
	RegMgr    registry.RegistryManager
	SbFactory sb.SandboxFactory
	PoolMgr   pmanager.PoolManager
	Config    *config.Config
	Lru       *HandlerLRU
}

// HandlerSet represents a collection of Handlers of a worker server. It
// manages the Handler by HandlerLRU.
type HandlerSet struct {
	mutex     sync.Mutex
	handlers  map[string]*Handler
	regMgr    registry.RegistryManager
	sbFactory sb.SandboxFactory
	poolMgr   pmanager.PoolManager
	config    *config.Config
	lru       *HandlerLRU
	workerDir string
	pipMirror string
}

// Handler handles requests to run a lambda on a worker server. It handles
// concurrency and communicates with the sandbox manager to change the
// state of the container that servers the lambda.
type Handler struct {
	mutex      sync.Mutex
	hset       *HandlerSet
	name       string
	sandbox    sb.Sandbox
	lastPull   *time.Time
	state      state.HandlerState
	runners    int
	code       []byte
	codeDir    string
	pkgs       []string
	sandboxDir string
}

// NewHandlerSet creates an empty HandlerSet
func NewHandlerSet(config *config.Config, lru *HandlerLRU) (handlerSet *HandlerSet, err error) {
	rm, err := registry.InitRegistryManager(config)
	if err != nil {
		return nil, err
	}

	sf, err := sb.InitSandboxFactory(config)
	if err != nil {
		return nil, err
	}

	pm, err := pmanager.InitPoolManager(config)
	if err != nil {
		return nil, err
	}

	handlerSet = &HandlerSet{
		handlers:  make(map[string]*Handler),
		regMgr:    rm,
		sbFactory: sf,
		poolMgr:   pm,
		lru:       lru,
		workerDir: config.Worker_dir,
		pipMirror: config.Pip_mirror,
	}

	return handlerSet, nil
}

// Get always returns a Handler, creating one if necessarily.
func (h *HandlerSet) Get(name string) *Handler {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	handler := h.handlers[name]
	if handler == nil {
		sandboxDir := path.Join(h.workerDir, "handlers", name, "sandbox")
		handler = &Handler{
			hset:       h,
			name:       name,
			state:      state.Unitialized,
			runners:    0,
			pkgs:       []string{},
			sandboxDir: sandboxDir,
		}
		h.handlers[name] = handler
	}

	return handler
}

// Dump prints the name and state of the Handlers currently in the HandlerSet.
func (h *HandlerSet) Dump() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	log.Printf("HANDLERS:\n")
	for k, v := range h.handlers {
		log.Printf("> %v: %v\n", k, v.state.String())
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
		codeDir, pkgs, err := h.hset.regMgr.Pull(h.name)
		if err != nil {
			return nil, err
		}

		now := time.Now()
		h.lastPull = &now
		h.codeDir = codeDir
		h.pkgs = pkgs
	}

	// create sandbox if needed
	if h.sandbox == nil {
		if err := os.MkdirAll(h.sandboxDir, 0666); err != nil {
			return nil, err
		}

		sandbox, err := h.hset.sbFactory.Create(h.codeDir, h.sandboxDir, h.hset.pipMirror)
		if err != nil {
			return nil, err
		}

		h.sandbox = sandbox
		if h.state, err = sandbox.State(); err != nil {
			return nil, err
		}

		// newly created sandbox could be in any state; let it run
		if h.state == state.Stopped {
			if err := sandbox.Start(); err != nil {
				return nil, err
			}
		} else if h.state == state.Paused {
			if err := sandbox.Unpause(); err != nil {
				return nil, err
			}
		}

		if h.hset.poolMgr == nil {
			containerSB, ok := h.sandbox.(sb.ContainerSandbox)
			if !ok {
				return nil, errors.New("pip packages only supported with ContainerSandbox")
			}
            if err := containerSB.DoInstalls(); err != nil {
                return nil, err
            }
		} else {
			containerSB, ok := h.sandbox.(sb.ContainerSandbox)
			if !ok {
				return nil, errors.New("forkenter only supported with ContainerSandbox")
			}

			if err := h.hset.poolMgr.Provision(containerSB, h.sandboxDir, h.pkgs); err != nil {
				return nil, err
			}
        }
	} else if h.state == state.Paused { // unpause if paused
		if err := h.sandbox.Unpause(); err != nil {
			return nil, err
		}
		h.hset.lru.Remove(h)
	}

	h.state = state.Running
	h.runners += 1

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
		h.state = state.Paused
		h.hset.lru.Add(h)
	}
}

// StopIfPaused stops the sandbox if it is paused.
func (h *Handler) StopIfPaused() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.state != state.Paused {
		return
	}

	// TODO(tyler): why do we need to unpause in order to kill?
	if err := h.sandbox.Unpause(); err != nil {
		log.Printf("Could not unpause %v to kill it!  Error: %v\n", h.name, err)
	} else if err := h.sandbox.Stop(); err != nil {
		// TODO: a resource leak?
		log.Printf("Could not kill %v after unpausing!  Error: %v\n", h.name, err)
	} else {
		h.state = state.Stopped
	}
}

// Sandbox returns the sandbox of this Handler.
func (h *Handler) Sandbox() sb.Sandbox {
	return h.sandbox
}

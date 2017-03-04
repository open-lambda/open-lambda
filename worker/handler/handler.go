// handler package implements a library for handling run lambda requests from
// the worker server.
package handler

import (
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler/state"
	"github.com/open-lambda/open-lambda/worker/registry"
	"github.com/open-lambda/open-lambda/worker/sandbox"

	pmanager "github.com/open-lambda/open-lambda/worker/pool-manager"
)

// HandlerSetOpts wraps parameters necessary to create a HandlerSet.
type HandlerSetOpts struct {
	Rm     registry.RegistryManager
	Sf     sandbox.SandboxFactory
	Pm     pmanager.PoolManager
	Config *config.Config
	Lru    *HandlerLRU
}

// HandlerSet represents a collection of Handlers of a worker server. It
// manages the Handler by HandlerLRU.
type HandlerSet struct {
	mutex    sync.Mutex
	handlers map[string]*Handler
	rm       registry.RegistryManager
	sf       sandbox.SandboxFactory
	pm       pmanager.PoolManager
	config   *config.Config
	lru      *HandlerLRU
}

// Handler handles requests to run a lambda on a worker server. It handles
// concurrency and communicates with the sandbox manager to change the
// state of the container that servers the lambda.
type Handler struct {
	mutex    sync.Mutex
	hset     *HandlerSet
	name     string
	sandbox  sandbox.Sandbox
	lastPull *time.Time
	state    state.HandlerState
	runners  int
	code     []byte
	codeDir  string
}

// NewHandlerSet creates an empty HandlerSet
func NewHandlerSet(opts HandlerSetOpts) (handlerSet *HandlerSet) {
	if opts.Lru == nil {
		opts.Lru = NewHandlerLRU(0)
	}

	return &HandlerSet{
		handlers: make(map[string]*Handler),
		rm:       opts.Rm,
		sf:       opts.Sf,
		pm:       opts.Pm,
		config:   opts.Config,
		lru:      opts.Lru,
	}
}

// Get always returns a Handler, creating one if necessarily.
func (h *HandlerSet) Get(name string) *Handler {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	handler := h.handlers[name]
	if handler == nil {
		handler = &Handler{
			hset:    h,
			name:    name,
			state:   state.Unitialized,
			runners: 0,
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
func (h *Handler) RunStart() (ch *sandbox.SandboxChannel, err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// get code if needed
	if h.lastPull == nil {
		codeDir, err := h.hset.rm.Pull(h.name)
		if err != nil {
			return nil, err
		}
		now := time.Now()
		h.lastPull = &now
		h.codeDir = codeDir
	}

	// create sandbox if needed
	if h.sandbox == nil {
		sandbox_dir := path.Join(h.hset.config.Worker_dir, "handlers", h.name, "sandbox")
		if err := os.MkdirAll(sandbox_dir, 0666); err != nil {
			return nil, err
		}

		sandbox, err := h.hset.sf.Create(h.codeDir, sandbox_dir)
		if err != nil {
			return nil, err
		}

		h.sandbox = sandbox
		h.state = state.Stopped
	}

	// are we the first?
	if h.runners == 0 {
		if h.state == state.Stopped {
			if err := h.sandbox.Start(); err != nil {
				return nil, err
			}

			// forkenter a handler server into sandbox if needed
			if h.hset.pm != nil {
				h.hset.pm.ForkEnter(h.sandbox)
			}
		} else if h.state == state.Paused {
			if err := h.sandbox.Unpause(); err != nil {
				return nil, err
			}
		}
		h.state = state.Running
		h.hset.lru.Remove(h)
	}

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
func (h *Handler) Sandbox() sandbox.Sandbox {
	return h.sandbox
}

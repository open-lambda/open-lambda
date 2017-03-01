package handler

import (
	"log"
	"os"
	"path"
	"sync"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/handler/state"
	"github.com/open-lambda/open-lambda/worker/sandbox"

	pmanager "github.com/open-lambda/open-lambda/worker/pool-manager"
	sbmanager "github.com/open-lambda/open-lambda/worker/sandbox-manager"
)

type HandlerSetOpts struct {
	Sm     sbmanager.SandboxManager
	Pm     pmanager.PoolManager
	Config *config.Config
	Lru    *HandlerLRU
}

type HandlerSet struct {
	mutex    sync.Mutex
	handlers map[string]*Handler
	sm       sbmanager.SandboxManager
	pm       pmanager.PoolManager
	config   *config.Config
	lru      *HandlerLRU
}

type Handler struct {
	mutex    sync.Mutex
	hset     *HandlerSet
	name     string
	sandbox  sandbox.Sandbox
	lastPull *time.Time
	state    state.HandlerState
	runners  int
	code     []byte
}

func NewHandlerSet(opts HandlerSetOpts) (handlerSet *HandlerSet) {
	if opts.Lru == nil {
		opts.Lru = NewHandlerLRU(0)
	}

	return &HandlerSet{
		handlers: make(map[string]*Handler),
		sm:       opts.Sm,
		pm:       opts.Pm,
		config:   opts.Config,
		lru:      opts.Lru,
	}
}

// always return a Handler, creating one if necessarily.
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

func (h *HandlerSet) Dump() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	log.Printf("HANDLERS:\n")
	for k, v := range h.handlers {
		log.Printf("> %v: %v\n", k, v.state.String())
	}
}

func (h *Handler) RunStart() (ch *sandbox.SandboxChannel, err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// get code if needed
	if h.lastPull == nil {
		err = h.hset.sm.Pull(h.name)
		if err != nil {
			return nil, err
		}
		now := time.Now()
		h.lastPull = &now
	}

	// create sandbox if needed
	if h.sandbox == nil {
		sandbox_dir := path.Join(h.hset.config.Worker_dir, "handlers", h.name, "sandbox")
		if err := os.MkdirAll(sandbox_dir, 0666); err != nil {
			return nil, err
		}

		sandbox, err := h.hset.sm.Create(h.name, sandbox_dir)
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

func (h *Handler) Sandbox() sandbox.Sandbox {
	return h.sandbox
}

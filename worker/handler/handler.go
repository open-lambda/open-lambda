package handler

import (
	"log"
	"sync"

	"github.com/open-lambda/open-lambda/worker/handler/state"
	"github.com/open-lambda/open-lambda/worker/sandbox"
)

type HandlerSetOpts struct {
	Cm  sandbox.SandboxManager
	Lru *HandlerLRU
}

type HandlerSet struct {
	mutex    sync.Mutex
	handlers map[string]*Handler
	cm       sandbox.SandboxManager
	lru      *HandlerLRU
}

type Handler struct {
	mutex   sync.Mutex
	hset    *HandlerSet
	name    string
	sandbox sandbox.Sandbox
	state   state.HandlerState
	runners int
}

func NewHandlerSet(opts HandlerSetOpts) (handlerSet *HandlerSet) {
	if opts.Lru == nil {
		opts.Lru = NewHandlerLRU(0)
	}

	return &HandlerSet{
		handlers: make(map[string]*Handler),
		cm:       opts.Cm,
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
			sandbox: h.cm.Create(name),
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

func (h *Handler) RunStart() (port string, err error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if err := h.maybeInit(); err != nil {
		return "", err
	}

	// are we the first?
	if h.runners == 0 {
		if h.state == state.Stopped {
			if err := h.sandbox.Start(); err != nil {
				return "", err
			}
		} else if h.state == state.Paused {
			if err := h.sandbox.Unpause(); err != nil {
				return "", err
			}
		}
		h.state = state.Running
		h.hset.lru.Remove(h)
	}

	h.runners += 1

	return h.sandbox.Port()
}

func (h *Handler) RunFinish() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.runners -= 1

	// are we the first?
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

// assume lock held.  Make sure image is pulled, and determine whether
// sandbox is running.
func (h *Handler) maybeInit() (err error) {
	if h.state != state.Unitialized {
		return nil
	}

	err = h.sandbox.MakeReady()
	if err != nil {
		return err
	}
	if h.state, err = h.sandbox.State(); err != nil {
		return err
	}

	return nil
}

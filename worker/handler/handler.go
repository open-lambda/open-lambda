package handler

import (
	"log"
	"sync"

	"github.com/tylerharter/open-lambda/worker/container"
	"github.com/tylerharter/open-lambda/worker/handler/state"
)

type HandlerSetOpts struct {
	Cm  container.ContainerManager
	Lru *HandlerLRU
}

type HandlerSet struct {
	mutex    sync.Mutex
	handlers map[string]*Handler
	cm       container.ContainerManager
	lru      *HandlerLRU
}

type Handler struct {
	mutex   sync.Mutex
	hset    *HandlerSet
	name    string
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

	cm := h.hset.cm

	// are we the first?
	if h.runners == 0 {
		if h.state == state.Stopped {
			if err := cm.Start(h.name); err != nil {
				return "", err
			}
		} else if h.state == state.Paused {
			if err := cm.Unpause(h.name); err != nil {
				return "", err
			}
		}
		h.state = state.Running
		h.hset.lru.Remove(h)
	}

	h.runners += 1

	info, err := cm.GetInfo(h.name)
	if err != nil {
		return "", err
	}

	return info.Port, nil
}

func (h *Handler) RunFinish() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	cm := h.hset.cm

	h.runners -= 1

	// are we the first?
	if h.runners == 0 {
		if err := cm.Pause(h.name); err != nil {
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

	cm := h.hset.cm

	if h.state != state.Paused {
		return
	}

	// TODO(tyler): why do we need to unpause in order to kill?
	if err := cm.Unpause(h.name); err != nil {
		log.Printf("Could not unpause %v to kill it!  Error: %v\n", h.name, err)
	} else if err := cm.Stop(h.name); err != nil {
		// TODO: a resource leak?
		log.Printf("Could not kill %v after unpausing!  Error: %v\n", h.name, err)
	} else {
		h.state = state.Stopped
	}
}

// assume lock held.  Make sure image is pulled, an determine whether
// container is running.
func (h *Handler) maybeInit() (err error) {
	if h.state != state.Unitialized {
		return nil
	}

	cm := h.hset.cm

	info, err := cm.MakeReady(h.name)
	if err != nil {
		return err
	}
	h.state = info.State

	return nil
}

package main

import (
	state "./handler_state"
	"sync"
)

type HandlerSet struct {
	mutex    sync.Mutex
	handlers map[string]*Handler
	cm       *ContainerManager
}

type Handler struct {
	mutex   sync.Mutex
	hset    *HandlerSet
	name    string
	state   state.HandlerState
	runners int
}

func NewHandlerSet(cm *ContainerManager) (handlerSet *HandlerSet) {
	return &HandlerSet{
		handlers: make(map[string]*Handler),
		cm:       cm,
	}
}

// always return a Handler, creating one if necessarily.
func (h *HandlerSet) Get(name string) *Handler {
	h.mutex.Lock()
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
	h.mutex.Unlock()

	return handler
}

func (h *Handler) RunStart() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if err := h.maybeInit(); err != nil {
		return err
	}

	h.runners += 1

	return nil
}

func (h *Handler) RunFinish() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.runners -= 1

	return nil
}

// assume lock held.  Make sure image is pulled, an determine whether
// container is running.
func (h *Handler) maybeInit() (err error) {
	if h.state != state.Unitialized {
		return nil
	}
	// if there is any error, set state back to Unitialized
	defer func() {
		if err != nil {
			h.state = state.Unitialized
		}
	}()

	// Is the image pulled?  Does container exist?  Is it paused?
	img_exists, err := h.hset.cm.DockerImageExists(h.name)
	if err != nil {
		return err
	}
	if img_exists {
		cont_exists, err := h.hset.cm.DockerContainerExists(h.name)
		if err != nil {
			return err
		}
		if cont_exists {
			// TODO(tyler): check if paused
			h.state = state.Running
		} else {
			h.state = state.Stopped
		}
	} else {
		if err := h.hset.cm.DockerPull(h.name); err != nil {
			return err
		}
		h.state = state.Stopped
	}
	return nil
}

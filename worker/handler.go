package main

import (
	state "github.com/tylerharter/open-lambda/worker/handler_state"
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
			if err := cm.DockerRestart(h.name); err != nil {
				return "", err
			}
		} else if h.state == state.Paused {
			if err := cm.DockerUnpause(h.name); err != nil {
				return "", err
			}
		}
	}

	h.runners += 1

	port, err = cm.getLambdaPort(h.name)
	if err != nil {
		return "", err
	}

	return port, nil
}

func (h *Handler) RunFinish() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.runners -= 1
}

// assume lock held.  Make sure image is pulled, an determine whether
// container is running.
func (h *Handler) maybeInit() (err error) {
	if h.state != state.Unitialized {
		return nil
	}

	cm := h.hset.cm

	// make sure image is pulled
	img_exists, err := cm.DockerImageExists(h.name)
	if err != nil {
		return err
	}
	if !img_exists {
		if err := cm.DockerPull(h.name); err != nil {
			return err
		}
	}

	// make sure container is created
	cont_exists, err := cm.DockerContainerExists(h.name)
	if err != nil {
		return err
	}
	if !cont_exists {
		if _, err := cm.DockerCreate(h.name, []string{}); err != nil {
			return err
		}
	}

	// is container stopped, running, or started?
	container, err := cm.DockerInspect(h.name)
	if err != nil {
		return err
	}

	if container.State.Running {
		h.state = state.Running
	} else if container.State.Paused {
		h.state = state.Paused
	} else {
		h.state = state.Stopped
	}

	return nil
}

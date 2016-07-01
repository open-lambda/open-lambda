package sandbox

import (
	"bytes"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/handler/state"
)

type DockerSandbox struct {
	name      string
	container *docker.Container
	mgr       *DockerManager
}

func (s *DockerSandbox) State() (hstate state.HandlerState, err error) {
	container, err := s.mgr.client.InspectContainer(s.container.ID)
	if err != nil {
		return hstate, err
	}

	if container.State.Running {
		if container.State.Paused {
			hstate = state.Paused
		} else {
			hstate = state.Running
		}
	} else {
		hstate = state.Stopped
	}
	return hstate, nil
}

func (s *DockerSandbox) Port() (port string, err error) {
	return s.mgr.getLambdaPort(s.container.ID)
}

// Starts a given container
func (s *DockerSandbox) Start() error {
	return s.mgr.dockerStart(s.container)
}

// Stops a given container
func (s *DockerSandbox) Stop() error {
	return s.mgr.dockerKill(s.container.ID)
}

// Pauses a given container
func (s *DockerSandbox) Pause() error {
	return s.mgr.dockerPause(s.container.ID)
}

// Unpauses a given container
func (s *DockerSandbox) Unpause() error {
	return s.mgr.dockerUnpause(s.container.ID)
}

// Frees all resources associated with a given lambda
// Will stop if needed
func (s *DockerSandbox) Remove() error {
	return s.mgr.dockerRemove(s.container)
}

// Return recent log output for container
func (s *DockerSandbox) Logs() (string, error) {
	buf := &bytes.Buffer{}
	if err := s.mgr.dockerLogs(s.container.ID, buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

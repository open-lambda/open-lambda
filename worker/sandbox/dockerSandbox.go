package sandbox

import (
	"bytes"

	"github.com/open-lambda/open-lambda/worker/handler/state"
)

type DockerSandbox struct {
	name string
	mgr  *DockerManager
}

// Runs any preperation to get the container ready to run
func (s *DockerSandbox) MakeReady() (err error) {
	// make sure image is pulled
	imgExists, err := s.mgr.DockerImageExists(s.name)
	if err != nil {
		return err
	}
	if !imgExists {
		if err := s.mgr.dockerPull(s.name); err != nil {
			return err
		}
	}

	// make sure container is created
	contExists, err := s.mgr.dockerContainerExists(s.name)
	if err != nil {
		return err
	}
	if !contExists {
		if _, err := s.mgr.dockerCreate(s.name, []string{}); err != nil {
			return err
		}
	}

	return nil
}

func (s *DockerSandbox) State() (hstate state.HandlerState, err error) {
	container, err := s.mgr.dockerInspect(s.name)
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
	return s.mgr.getLambdaPort(s.name)
}

// Starts a given container
func (s *DockerSandbox) Start() error {
	c, err := s.mgr.dockerInspect(s.name)
	if err != nil {
		return err
	}

	return s.mgr.dockerStart(c)
}

// Pauses a given container
func (s *DockerSandbox) Pause() error {
	return s.mgr.dockerPause(s.name)
}

// Unpauses a given container
func (s *DockerSandbox) Unpause() error {
	return s.mgr.dockerUnpause(s.name)
}

// Stops a given container
func (s *DockerSandbox) Stop() error {
	return s.mgr.dockerKill(s.name)
}

// Frees all resources associated with a given lambda
// Will stop if needed
func (s *DockerSandbox) Remove() error {
	c, err := s.mgr.dockerInspect(s.name)
	if err != nil {
		return s.mgr.dockerError(s.name, err)
	}

	return s.mgr.dockerRemove(c)
}

// Return recent log output for container
func (s *DockerSandbox) Logs() (string, error) {
	container, err := s.mgr.dockerInspect(s.name)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	if err := s.mgr.dockerLogs(container.ID, buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

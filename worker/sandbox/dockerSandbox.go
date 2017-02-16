/*

Provides the mechanism for managing a given Docker container-based lambda.

Must be paired with a DockerSandboxManager which handles pulling handler
code, initializing containers, etc.

*/

package sandbox

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/handler/state"
)

type DockerSandbox struct {
	name        string
	sandbox_dir string
        nspid       int
	container   *docker.Container
	client      *docker.Client
}

func NewDockerSandbox(name string, sandbox_dir string, nspid int, container *docker.Container, client *docker.Client) *DockerSandbox {
	sandbox := &DockerSandbox{
		name:        name,
		sandbox_dir: sandbox_dir,
                nspid:       nspid,
		container:   container,
		client:      client,
	}
	return sandbox
}

func (s *DockerSandbox) dockerError(outer error) (err error) {
	buf := bytes.NewBufferString(outer.Error() + ".  ")

	if err := s.InspectUpdate(); err != nil {
		buf.WriteString(fmt.Sprintf("Could not inspect container (%v).  ", err.Error()))
	} else {
		buf.WriteString(fmt.Sprintf("Container state is <%v>.  ", s.container.State.StateString()))
	}

	if log, err := s.Logs(); err != nil {
		buf.WriteString(fmt.Sprintf("Could not fetch [%s] logs!\n", s.container.ID))
	} else {
		buf.WriteString(fmt.Sprintf("<--- Start handler container [%s] logs: --->\n", s.container.ID))
		buf.WriteString(log)
		buf.WriteString(fmt.Sprintf("<--- End handler container [%s] logs --->\n", s.container.ID))
	}

	return errors.New(buf.String())
}

func (s *DockerSandbox) InspectUpdate() error {
	container, err := s.client.InspectContainer(s.container.ID)
	if err != nil {
		return err
	}
	s.container = container

	return nil
}

func (s *DockerSandbox) State() (hstate state.HandlerState, err error) {
	if err := s.InspectUpdate(); err != nil {
		return hstate, err
	}

	if s.container.State.Running {
		if s.container.State.Paused {
			hstate = state.Paused
		} else {
			hstate = state.Running
		}
	} else {
		hstate = state.Stopped
	}

	return hstate, nil
}

func (s *DockerSandbox) Channel() (channel *SandboxChannel, err error) {
	if err := s.InspectUpdate(); err != nil {
		return nil, s.dockerError(err)
	}

	var env docker.Env
	env = s.container.Config.Env

	if env.Exists("ol.config") {
		var conf map[string]interface{}
		if err := json.Unmarshal([]byte(env.Get("ol.config")), &conf); err != nil {
			return nil, err
		}

		if val, exists := conf["sock_file"]; exists {
			switch val := val.(type) {
			default:
				return nil, fmt.Errorf("sock_file must be a string")
			case string:
				dial := func(proto, addr string) (net.Conn, error) {
					return net.Dial("unix", path.Join(s.sandbox_dir, val))
				}
				tr := http.Transport{Dial: dial}

				// the server name doesn't matter since we have a sock file
				return &SandboxChannel{Url: "http://container", Transport: tr}, nil
			}
		}
	}

	container_port := docker.Port("8080/tcp")
	ports := s.container.NetworkSettings.Ports[container_port]
	if len(ports) == 0 {
		err := fmt.Errorf("could not lookup host port for %v", container_port)
		return nil, s.dockerError(err)
	} else if len(ports) > 1 {
		err := fmt.Errorf("multiple host port mapping to %v", container_port)
		return nil, s.dockerError(err)
	}
	port := ports[0].HostPort

	// on unix systems, port is given as "unix:port", this removes the prefix
	if strings.HasPrefix(port, "unix") {
		port = strings.Split(port, ":")[1]
	}

	url := fmt.Sprintf("http://localhost:%s", port)
	return &SandboxChannel{Url: url}, nil
}

/* Starts the container */
func (s *DockerSandbox) Start() error {
	if err := s.client.StartContainer(s.container.ID, s.container.HostConfig); err != nil {
		log.Printf("failed to start container with err %v\n", err)
		return s.dockerError(err)
	}

	return nil
}

/* Stops the container */
func (s *DockerSandbox) Stop() error {
	// TODO(tyler): is there any advantage to trying to stop
	// before killing?  (i.e., use SIGTERM instead SIGKILL)
	opts := docker.KillContainerOptions{ID: s.container.ID}
	if err := s.client.KillContainer(opts); err != nil {
		log.Printf("failed to kill container with error %v\n", err)
		return s.dockerError(err)
	}

	return nil
}

/* Pauses the container */
func (s *DockerSandbox) Pause() error {

	if err := s.client.PauseContainer(s.container.ID); err != nil {
		log.Printf("failed to pause container with error %v\n", err)
		return s.dockerError(err)
	}

	return nil
}

/* Unpauses the container */
func (s *DockerSandbox) Unpause() error {
	if err := s.client.UnpauseContainer(s.container.ID); err != nil {
		log.Printf("failed to unpause container %s with err %v\n", s.name, err)
		return s.dockerError(err)
	}

	return nil
}

/* Frees all resources associated with the lambda (stops the container if necessary) */
func (s *DockerSandbox) Remove() error {
	if err := s.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: s.container.ID,
	}); err != nil {
		log.Printf("failed to rm container with err %v", err)
		return s.dockerError(err)
	}

	return nil
}

/* Return log output for the container */
func (s *DockerSandbox) Logs() (string, error) {
	buf := &bytes.Buffer{}
	err := s.client.Logs(docker.LogsOptions{
		Container:         s.container.ID,
		OutputStream:      buf,
		ErrorStream:       buf,
		InactivityTimeout: time.Second,
		Follow:            false,
		Stdout:            true,
		Stderr:            true,
		Since:             0,
		Timestamps:        false,
		Tail:              "20",
		RawTerminal:       false,
	})

	if err != nil {
		log.Printf("failed to get logs for %s with err %v\n", s.name, err)
		return "", err
	}

	return buf.String(), nil
}

/*

Provides the mechanism for managing a given Docker container-based lambda.

Must be paired with a DockerSandboxManager which handles pulling handler
code, initializing containers, etc.

*/

package sandbox

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/benchmarker"
	"github.com/open-lambda/open-lambda/worker/handler/state"
)

// DockerSandbox is a sandbox inside a docker container.
type DockerSandbox struct {
	sandbox_dir string
	nspid       string
	container   *docker.Container
	client      *docker.Client
	tr          http.Transport
	installed   map[string]bool
	index_host  string
	index_port  string
}

// NewDockerSandbox creates a DockerSandbox.
func NewDockerSandbox(sandbox_dir, index_host, index_port string, container *docker.Container, client *docker.Client) *DockerSandbox {
	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", filepath.Join(sandbox_dir, "ol.sock"))
	}
	tr := http.Transport{Dial: dial, DisableKeepAlives: true}

	sandbox := &DockerSandbox{
		sandbox_dir: sandbox_dir,
		container:   container,
		client:      client,
		tr:          tr,
		installed:   make(map[string]bool),
		index_host:  index_host,
		index_port:  index_port,
	}

	return sandbox
}

// dockerError adds details (sandbox log, state, etc.) to an error.
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

// InspectUpdate calls docker inspect to update the state of the container.
func (s *DockerSandbox) InspectUpdate() error {
	container, err := s.client.InspectContainer(s.container.ID)
	if err != nil {
		return err
	}
	s.container = container

	return nil
}

// State returns the state of the Docker sandbox.
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

// Channel returns a file socket channel for direct communication with the sandbox.
func (s *DockerSandbox) Channel() (channel *SandboxChannel, err error) {
	if err := s.InspectUpdate(); err != nil {
		return nil, s.dockerError(err)
	}
	// the server name doesn't matter since we have a sock file
	return &SandboxChannel{Url: "http://container", Transport: s.tr}, nil
}

// Start starts the container.
func (s *DockerSandbox) Start() error {
	b := benchmarker.GetBenchmarker()
	var t *benchmarker.Timer
	if b != nil {
		t = b.CreateTimer("Start docker container", "ms")
		t.Start()
	}

	if err := s.client.StartContainer(s.container.ID, nil); err != nil {
		log.Printf("failed to start container with err %v\n", err)
		if t != nil {
			t.Error("Failed to start docker container")
		}
		return s.dockerError(err)
	}

	if t != nil {
		t.End()
	}

	container, err := s.client.InspectContainer(s.container.ID)
	if err != nil {
		log.Printf("failed to inpect container with err %v\n", err)
		return s.dockerError(err)
	}
	s.container = container
	s.nspid = fmt.Sprintf("%d", container.State.Pid)

	return nil
}

// Stop stops the container.
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

// Pause pauses the container.
func (s *DockerSandbox) Pause() error {
	b := benchmarker.GetBenchmarker()
	var t *benchmarker.Timer
	if b != nil {
		t = b.CreateTimer("Pause docker container", "ms")
		t.Start()
	}
	if err := s.client.PauseContainer(s.container.ID); err != nil {
		log.Printf("failed to pause container with error %v\n", err)
		if t != nil {
			t.Error("Failed to pause docker container")
		}
		return s.dockerError(err)
	}
	if t != nil {
		t.End()
	}
	return nil
}

// Unpause unpauses the container.
func (s *DockerSandbox) Unpause() error {
	b := benchmarker.GetBenchmarker()
	var t *benchmarker.Timer
	if b != nil {
		t = b.CreateTimer("Unpause docker container", "ms")
		t.Start()
	}

	if err := s.client.UnpauseContainer(s.container.ID); err != nil {
		log.Printf("failed to unpause container %s with err %v\n", s.container.Name, err)
		if t != nil {
			t.Error("Failed to unpause docker container")
		}
		return s.dockerError(err)
	}

	if t != nil {
		t.End()
	}

	return nil
}

// Remove frees all resources associated with the lambda (stops the container if necessary).
func (s *DockerSandbox) Remove() error {
	if err := s.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: s.container.ID,
	}); err != nil {
		log.Printf("failed to rm container with err %v", err)
		return s.dockerError(err)
	}

	return nil
}

// Logs returns log output for the container.
func (s *DockerSandbox) Logs() (string, error) {
	stdout_path := filepath.Join(s.sandbox_dir, "stdout")
	stderr_path := filepath.Join(s.sandbox_dir, "stderr")

	stdout, err := ioutil.ReadFile(stdout_path)
	if err != nil {
		return "", err
	}

	stderr, err := ioutil.ReadFile(stderr_path)
	if err != nil {
		return "", err
	}

	stdout_hdr := fmt.Sprintf("Container (%s) stdout:", s.container.ID)
	stderr_hdr := fmt.Sprintf("Container (%s) stderr:", s.container.ID)
	ret := fmt.Sprintf("%s\n%s\n%s\n%s\n", stdout_hdr, stdout, stderr_hdr, stderr)

	return ret, nil
}

// Put the passed process into the cgroup of this docker container.
func (s *DockerSandbox) CGroupEnter(pid string) (err error) {
	b := benchmarker.GetBenchmarker()
	var t *benchmarker.Timer
	if b != nil {
		t = b.CreateTimer("cgclassify process into docker container", "us")
	}

	controllers := "memory,cpu,devices,perf_event,cpuset,blkio,pids,freezer,net_cls,net_prio,hugetlb"
	cgroupArg := fmt.Sprintf("%s:/docker/%s", controllers, s.container.ID)
	cmd := exec.Command("cgclassify", "--sticky", "-g", cgroupArg, pid)

	if t != nil {
		t.Start()
	}

	if err := cmd.Run(); err != nil {
		if t != nil {
			t.Error("Failed to run cgclassify")
		}
		return err
	}

	if t != nil {
		t.End()
	}

	return nil
}

// NSPid returns the pid of the first process of the docker container.
func (s *DockerSandbox) NSPid() string {
	return s.nspid
}

func (s *DockerSandbox) ID() string {
	return s.container.ID
}

func (s *DockerSandbox) RunServer() error {
	cmd := []string{"python", "server.py"}
	if s.index_host != "" && s.index_port != "" {
		cmd = append(cmd, s.index_host, s.index_port)
	}

	execOpts := docker.CreateExecOptions{
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Container:    s.container.ID,
		Cmd:          cmd,
	}

	if exec, err := s.client.CreateExec(execOpts); err != nil {
		return err
	} else if err := s.client.StartExec(exec.ID, docker.StartExecOptions{}); err != nil {
		return err
	}

	return nil
}

func (s *DockerSandbox) MemoryCGroupPath() string {
	return fmt.Sprintf("/sys/fs/cgroup/memory/docker/%s/", s.container.ID)
}

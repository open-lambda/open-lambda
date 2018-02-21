/*

Provides the mechanism for managing a given Docker container-based lambda.

Must be paired with a DockerContainerManager which handles pulling handler
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
	"os"
	"os/exec"
	"path/filepath"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/benchmarker"
	"github.com/open-lambda/open-lambda/worker/handler/state"
)

// DockerContainer is a sandbox inside a docker container.
type DockerContainer struct {
	host_id   string
	hostDir   string
	nspid     string
	container *docker.Container
	client    *docker.Client
	installed map[string]bool
	cache     bool
}

// NewDockerContainer creates a DockerContainer.
func NewDockerContainer(host_id, hostDir string, cache bool, container *docker.Container, client *docker.Client) *DockerContainer {
	sandbox := &DockerContainer{
		host_id:   host_id,
		hostDir:   hostDir,
		container: container,
		client:    client,
		installed: make(map[string]bool),
		cache:     cache,
	}

	return sandbox
}

// dockerError adds details (sandbox log, state, etc.) to an error.
func (c *DockerContainer) dockerError(outer error) (err error) {
	buf := bytes.NewBufferString(outer.Error() + ".  ")

	if err := c.InspectUpdate(); err != nil {
		buf.WriteString(fmt.Sprintf("Could not inspect container (%v).  ", err.Error()))
	} else {
		buf.WriteString(fmt.Sprintf("Container state is <%v>.  ", c.container.State.StateString()))
	}

	if log, err := c.Logs(); err != nil {
		buf.WriteString(fmt.Sprintf("Could not fetch [%s] logs!\n", c.container.ID))
	} else {
		buf.WriteString(fmt.Sprintf("<--- Start handler container [%s] logs: --->\n", c.container.ID))
		buf.WriteString(log)
		buf.WriteString(fmt.Sprintf("<--- End handler container [%s] logs --->\n", c.container.ID))
	}

	return errors.New(buf.String())
}

// InspectUpdate calls docker inspect to update the state of the container.
func (c *DockerContainer) InspectUpdate() error {
	container, err := c.client.InspectContainer(c.container.ID)
	if err != nil {
		return err
	}
	c.container = container

	return nil
}

// State returns the state of the Docker sandbox.
func (c *DockerContainer) State() (hstate state.HandlerState, err error) {
	if err := c.InspectUpdate(); err != nil {
		return hstate, err
	}

	if c.container.State.Running {
		if c.container.State.Paused {
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
func (c *DockerContainer) Channel() (channel *Channel, err error) {
	sockPath := filepath.Join(c.hostDir, "ol.sock")
	if len(sockPath) > 108 {
		return nil, fmt.Errorf("socket path length cannot exceed 108 characters (try moving cluster closer to the root directory")
	}

	dial := func(proto, addr string) (net.Conn, error) {
		return net.Dial("unix", sockPath)
	}
	tr := http.Transport{Dial: dial}

	// the server name doesn't matter since we have a sock file
	return &Channel{Url: "http://container/", Transport: tr}, nil
}

// Start starts the container.
func (c *DockerContainer) Start() error {
	b := benchmarker.GetBenchmarker()
	var t *benchmarker.Timer
	if b != nil {
		t = b.CreateTimer("Start docker container", "ms")
		t.Start()
	}

	if err := c.client.StartContainer(c.container.ID, nil); err != nil {
		log.Printf("failed to start container with err %v\n", err)
		if t != nil {
			t.Error("Failed to start docker container")
		}
		return c.dockerError(err)
	}

	if t != nil {
		t.End()
	}

	container, err := c.client.InspectContainer(c.container.ID)
	if err != nil {
		log.Printf("failed to inpect container with err %v\n", err)
		return c.dockerError(err)
	}
	c.container = container
	c.nspid = fmt.Sprintf("%d", container.State.Pid)

	return nil
}

// Stop stops the container.
func (c *DockerContainer) Stop() error {
	// TODO(tyler): is there any advantage to trying to stop
	// before killing?  (i.e., use SIGTERM instead SIGKILL)
	opts := docker.KillContainerOptions{ID: c.container.ID}
	if err := c.client.KillContainer(opts); err != nil {
		log.Printf("failed to kill container with error %v\n", err)
		return c.dockerError(err)
	}

	return nil
}

// Pause pauses the container.
func (c *DockerContainer) Pause() error {
	b := benchmarker.GetBenchmarker()
	var t *benchmarker.Timer
	if b != nil {
		t = b.CreateTimer("Pause docker container", "ms")
		t.Start()
	}
	if err := c.client.PauseContainer(c.container.ID); err != nil {
		log.Printf("failed to pause container with error %v\n", err)
		if t != nil {
			t.Error("Failed to pause docker container")
		}
		return c.dockerError(err)
	}
	if t != nil {
		t.End()
	}
	return nil
}

// Unpause unpauses the container.
func (c *DockerContainer) Unpause() error {
	b := benchmarker.GetBenchmarker()
	var t *benchmarker.Timer
	if b != nil {
		t = b.CreateTimer("Unpause docker container", "ms")
		t.Start()
	}

	if err := c.client.UnpauseContainer(c.container.ID); err != nil {
		log.Printf("failed to unpause container %s with err %v\n", c.container.Name, err)
		if t != nil {
			t.Error("Failed to unpause docker container")
		}
		return c.dockerError(err)
	}

	if t != nil {
		t.End()
	}

	return nil
}

// Remove frees all resources associated with the lambda (stops the container if necessary).
func (c *DockerContainer) Remove() error {
	// remove sockets if they exist
	if err := os.RemoveAll(filepath.Join(c.hostDir, "ol.sock")); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(c.hostDir, "fs.sock")); err != nil {
		return err
	}

	if err := c.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: c.container.ID,
	}); err != nil {
		log.Printf("failed to rm container with err %v", err)
		return c.dockerError(err)
	}

	return nil
}

// Logs returns log output for the container.
func (c *DockerContainer) Logs() (string, error) {
	stdout_path := filepath.Join(c.hostDir, "stdout")
	stderr_path := filepath.Join(c.hostDir, "stderr")

	stdout, err := ioutil.ReadFile(stdout_path)
	if err != nil {
		return "", err
	}

	stderr, err := ioutil.ReadFile(stderr_path)
	if err != nil {
		return "", err
	}

	stdout_hdr := fmt.Sprintf("Container (%s) stdout:", c.container.ID)
	stderr_hdr := fmt.Sprintf("Container (%s) stderr:", c.container.ID)
	ret := fmt.Sprintf("%s\n%s\n%s\n%s\n", stdout_hdr, stdout, stderr_hdr, stderr)

	return ret, nil
}

// Put the passed process into the cgroup of this docker container.
func (c *DockerContainer) CGroupEnter(pid string) (err error) {
	b := benchmarker.GetBenchmarker()
	var t *benchmarker.Timer
	if b != nil {
		t = b.CreateTimer("cgclassify process into docker container", "us")
	}

	controllers := "memory,cpu,devices,perf_event,cpuset,blkio,pids,freezer,net_cls,net_prio,hugetlb"
	cgroupArg := fmt.Sprintf("%s:/docker/%s", controllers, c.container.ID)
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
func (c *DockerContainer) NSPid() string {
	return c.nspid
}

func (c *DockerContainer) ID() string {
	return c.host_id
}

func (c *DockerContainer) DockerID() string {
	return c.container.ID
}

func (c *DockerContainer) RunServer() error {
	cmd := []string{"python", "server.py"}
	if c.cache {
		cmd = append(cmd, "--cache")
	}

	execOpts := docker.CreateExecOptions{
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Container:    c.container.ID,
		Cmd:          cmd,
	}

	if exec, err := c.client.CreateExec(execOpts); err != nil {
		return err
	} else if err := c.client.StartExec(exec.ID, docker.StartExecOptions{}); err != nil {
		return err
	}

	return nil
}

func (c *DockerContainer) MemoryCGroupPath() string {
	return fmt.Sprintf("/sys/fs/cgroup/memory/docker/%s/", c.container.ID)
}

func (c *DockerContainer) RootDir() string {
	return "/"
}

func (c *DockerContainer) HostDir() string {
	return c.hostDir
}

func (c *DockerContainer) Pipe() *os.File {
	return nil
}

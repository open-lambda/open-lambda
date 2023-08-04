/*

Provides the mechanism for managing a given Docker container-based lambda.

Must be paired with a DockerContainerManager which handles pulling handler
code, initializing containers, etcontainer.

*/

package sandbox

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"net/http"
	"path/filepath"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/ol/common"
)

// DockerContainer is a sandbox inside a docker container.
type DockerContainer struct {
	hostID	string
	hostDir   string
	nspid	 string
	container *docker.Container
	client	*docker.Client
	installed map[string]bool
	meta	  *SandboxMeta
	rtType   common.RuntimeType
	httpClient *http.Client
}

type HandlerState int

const (
	Unitialized HandlerState = iota
	Running
	Paused
)

func (h HandlerState) String() string {
	switch h {
	case Unitialized:
		return "unitialized"
	case Running:
		return "running"
	case Paused:
		return "paused"
	default:
		panic("Unknown state!")
	}
}

// dockerError adds details (sandbox log, state, etcontainer.) to an error.
func (container *DockerContainer) dockerError(outer error) (err error) {
	buf := bytes.NewBufferString(outer.Error() + ".  ")

	if err := container.InspectUpdate(); err != nil {
		buf.WriteString(fmt.Sprintf("Could not inspect container (%v).  ", err.Error()))
	} else {
		buf.WriteString(fmt.Sprintf("Container state is <%v>.  ", container.container.State.StateString()))
	}

	if logMsg, err := container.Logs(); err != nil {
		buf.WriteString(fmt.Sprintf("Could not fetch [%s] logs!\n", container.container.ID))
	} else {
		buf.WriteString(fmt.Sprintf("<--- Start handler container [%s] logs: --->\n", container.container.ID))
		buf.WriteString(logMsg)
		buf.WriteString(fmt.Sprintf("<--- End handler container [%s] logs --->\n", container.container.ID))
	}

	return errors.New(buf.String())
}

// InspectUpdate calls docker inspect to update the state of the container.
func (container *DockerContainer) InspectUpdate() error {
	inspect, err := container.client.InspectContainer(container.container.ID)
	if err != nil {
		return err
	}
	container.container = inspect

	return nil
}

// State returns the state of the Docker sandbox.
func (container *DockerContainer) State() (hstate HandlerState, err error) {
	if err := container.InspectUpdate(); err != nil {
		return hstate, err
	}

	if container.container.State.Running {
		if container.container.State.Paused {
			hstate = Paused
		} else {
			hstate = Running
		}
	} else {
		return hstate, fmt.Errorf("unexpected state")
	}

	return hstate, nil
}

func (container *DockerContainer) Client() (*http.Client) {
	return container.httpClient
}

// Start starts the container.
func (container *DockerContainer) start() error {
	if err := container.client.StartContainer(container.container.ID, nil); err != nil {
		log.Printf("failed to start container with err %v\n", err)
		return container.dockerError(err)
	}

	inspect, err := container.client.InspectContainer(container.container.ID)
	if err != nil {
		log.Printf("failed to inpect container with err %v\n", err)
		return container.dockerError(err)
	}
	container.container = inspect
	container.nspid = fmt.Sprintf("%d", inspect.State.Pid)

	return nil
}

// Pause stops/freezes the container.
func (container *DockerContainer) Pause() error {
	st, err := container.State()
	if err != nil {
		return err
	} else if st == Paused {
		return nil
	}

	if err := container.client.PauseContainer(container.container.ID); err != nil {
		log.Printf("failed to pause container with error %v\n", err)
		return container.dockerError(err)
	}

	// idle connections use a LOT of memory in the OL process
	container.httpClient.CloseIdleConnections()

	return nil
}

// Unpause resumes/unfreezes the container.
func (container *DockerContainer) Unpause() error {
	st, err := container.State()
	if err != nil {
		return err
	} else if st == Running {
		return nil
	}

	if err := container.client.UnpauseContainer(container.container.ID); err != nil {
		log.Printf("failed to unpause container %s with err %v\n", container.container.Name, err)
		return container.dockerError(err)
	}

	return nil
}

// Destroy shuts down this container
func (container *DockerContainer) Destroy(reason string) {
	if err := container.internalDestroy(); err != nil {
		panic(fmt.Sprintf("Failed to cleanup container %v: %v", container.container.ID, err))
	}
}

func (container *DockerContainer) DestroyIfPaused(reason string) {
	container.Destroy(reason) // we're allowed to implement this by uncondationally destroying
}

// frees all resources associated with the lambda
func (container *DockerContainer) internalDestroy() error {
	container.Unpause()

	// TODO(tyler): is there any advantage to trying to stop
	// before killing?  (i.e., use SIGTERM instead SIGKILL)
	opts := docker.KillContainerOptions{ID: container.container.ID}
	if err := container.client.KillContainer(opts); err != nil {
		log.Printf("failed to kill container with error %v\n", err)
		return container.dockerError(err)
	}

	// remove sockets if they exist
	if err := os.RemoveAll(filepath.Join(container.hostDir, "ol.sock")); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(container.hostDir, "fs.sock")); err != nil {
		return err
	}

	if err := container.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: container.container.ID,
	}); err != nil {
		log.Printf("failed to rm container with err %v", err)
		return container.dockerError(err)
	}

	return nil
}

// Logs returns log output for the container.
func (container *DockerContainer) Logs() (string, error) {
	stdoutPath := filepath.Join(container.hostDir, "stdout")
	stderrPath := filepath.Join(container.hostDir, "stderr")

	stdout, err := ioutil.ReadFile(stdoutPath)
	if err != nil {
		return "", err
	}

	stderr, err := ioutil.ReadFile(stderrPath)
	if err != nil {
		return "", err
	}

	stdoutHdr := fmt.Sprintf("Container (%s) stdout:", container.container.ID)
	stderrHdr := fmt.Sprintf("Container (%s) stderr:", container.container.ID)
	ret := fmt.Sprintf("%s\n%s\n%s\n%s\n", stdoutHdr, stdout, stderrHdr, stderr)

	return ret, nil
}

// GetRuntimeLog returns the log of the runtime
// Note, this is not supported for docker yet
func (*DockerContainer) GetRuntimeLog() string {
	return "" //TODO
}

// GetProxyLog returns the log of the http proxy
// Note, this is not supported for docker yet
func (*DockerContainer) GetProxyLog() string {
	return "" //TODO
}

// NSPid returns the pid of the first process of the docker container.
func (container *DockerContainer) NSPid() string {
	return container.nspid
}

// ID returns the identifier of this container
func (container *DockerContainer) ID() string {
	return container.hostID
}

// GetRuntimeType returns what runtime is being used by this container?
func (container *DockerContainer) GetRuntimeType() common.RuntimeType {
	return container.rtType
}

// DockerID returns the id assigned by docker itself, not by open lambda
func (container *DockerContainer) DockerID() string {
	return container.container.ID
}

// HostDir returns the host directory of this container
func (container *DockerContainer) HostDir() string {
	return container.hostDir
}

func (container *DockerContainer) runServer() error {
	if container.rtType != common.RT_PYTHON {
		return fmt.Errorf("Unsupported runtime")
	}

	cmd := []string{"python3", "/runtimes/python/server_legacy.py"}

	execOpts := docker.CreateExecOptions{
		AttachStdin:  false,
		AttachStdout: false,
		AttachStderr: false,
		Container:    container.container.ID,
		Cmd:          cmd,
	}

	if exec, err := container.client.CreateExec(execOpts); err != nil {
		return err
	} else if err := container.client.StartExec(exec.ID, docker.StartExecOptions{}); err != nil {
		return err
	}

	return nil
}

func (container *DockerContainer) DebugString() string {
	return fmt.Sprintf("SANDBOX %s (DOCKER)\n", container.ID())
}

func (*DockerContainer) fork(dst Sandbox) (err error) {
	panic("DockerContainer does not implement cross-container forks")
}

func (*DockerContainer) childExit(child Sandbox) {
	panic("DockerContainers should not have children because fork is unsupported")
}

func waitForServerPipeReady(hostDir string) error {
	// upon success, the goroutine will send nil; else, it will send the error
	ready := make(chan error, 1)

	go func() {
		pipeFile := filepath.Join(hostDir, "server_pipe")
		pipe, err := os.OpenFile(pipeFile, os.O_RDWR, 0777)
		if err != nil {
			log.Printf("Cannot open pipe: %v\n", err)
			return
		}
		defer pipe.Close()

		// wait for "ready"
		buf := make([]byte, 5)
		_, err = pipe.Read(buf)
		if err != nil {
			ready <- fmt.Errorf("cannot read from stdout of sandbox :: %v", err)
		} else if string(buf) != "ready" {
			ready <- fmt.Errorf("expect to see `ready` but got %s", string(buf))
		}
		ready <- nil
	}()

	// TODO: make timeout configurable
	timeout := time.NewTimer(20 * time.Second)
	defer timeout.Stop()

	select {
	case err := <-ready:
		return err
	case <-timeout.C:
		return fmt.Errorf("instance server failed to initialize after 20s")
	}
}

func (container *DockerContainer) Meta() *SandboxMeta {
	return container.meta
}

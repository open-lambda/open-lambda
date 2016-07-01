package sandbox

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/phonyphonecall/turnip"
)

type DockerManager struct {
	client *docker.Client

	registryName string
	opts         *config.Config

	// timers
	createTimer  *turnip.Turnip
	pauseTimer   *turnip.Turnip
	unpauseTimer *turnip.Turnip
	pullTimer    *turnip.Turnip
	restartTimer *turnip.Turnip
	inspectTimer *turnip.Turnip
	startTimer   *turnip.Turnip
	removeTimer  *turnip.Turnip
	logTimer     *turnip.Turnip
}

func NewDockerManager(opts *config.Config) (manager *DockerManager) {
	manager = new(DockerManager)

	// NOTE: This requires that users have pre-configured the environement a docker daemon
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		manager.client = c
	}

	manager.opts = opts
	manager.registryName = fmt.Sprintf("%s:%s", opts.Registry_host, opts.Registry_port)
	manager.initTimers()
	return manager
}

func (dm *DockerManager) Create(name string) Sandbox {
	return &DockerSandbox{name: name, mgr: dm}
}

func (dm *DockerManager) Pull(name string) error {
	// delete if it exists, so we can pull a new one
	imgExists, err := dm.DockerImageExists(name)
	if err != nil {
		return err
	}
	if imgExists {
		if dm.opts.Skip_pull_existing {
			return nil
		}
		opts := docker.RemoveImageOptions{Force: true}
		if err := dm.client.RemoveImageExtended(name, opts); err != nil {
			return err
		}
	}

	// pull new code
	if err := dm.dockerPull(name); err != nil {
		return err
	}

	return nil
}

func (dm *DockerManager) dockerLogs(cid string, buf *bytes.Buffer) (err error) {
	dm.logTimer.Start()

	err = dm.client.Logs(docker.LogsOptions{
		Container:         cid,
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
		log.Printf("failed to get logs for %s with err %v\n", cid, err)
		return err
	}

	dm.logTimer.Stop()

	return nil
}

func (dm *DockerManager) dockerError(cid string, outer error) (err error) {
	buf := bytes.NewBufferString(outer.Error() + ".  ")

	container, err := dm.dockerInspect(cid)
	if err != nil {
		buf.WriteString(fmt.Sprintf("Could not inspect container (%v).  ", err.Error()))
	} else {
		buf.WriteString(fmt.Sprintf("Container state is <%v>.  ", container.State.StateString()))
	}

	buf.WriteString(fmt.Sprintf("<--- Start handler container [%s] logs: --->\n", cid))

	err = dm.dockerLogs(cid, buf)
	if err != nil {
		return err
	}

	buf.WriteString(fmt.Sprintf("<--- End handler container [%s] logs --->\n", cid))

	return errors.New(buf.String())
}

func (dm *DockerManager) pullAndCreate(img string, args []string) (container *docker.Container, err error) {
	if container, err = dm.dockerCreate(img, args); err != nil {
		// if the container already exists, don't pull, let client decide how to handle
		if err == docker.ErrContainerAlreadyExists {
			return nil, err
		}

		if err = dm.dockerPull(img); err != nil {
			log.Printf("img pull failed with: %v\n", err)
			return nil, err
		} else {
			container, err = dm.dockerCreate(img, args)
			if err != nil {
				log.Printf("failed to create container %s after good pull, with error: %v\n", img, err)
				return nil, err
			}
		}
	}

	return container, nil
}

// Will ensure given image is running
// returns the port of the runnning container
func (dm *DockerManager) dockerMakeReady(img string) (port string, err error) {
	// TODO: decide on one default lambda entry path
	container, err := dm.pullAndCreate(img, []string{})
	if err != nil {
		if err != docker.ErrContainerAlreadyExists {
			// Unhandled error
			return "", err
		}

		// make sure container is up
		cid := img
		container, err = dm.dockerInspect(cid)
		if err != nil {
			return "", err
		}
		if container.State.Paused {
			// unpause
			if err = dm.dockerUnpause(container.ID); err != nil {
				return "", err
			}
		} else if !container.State.Running {
			// restart a stopped/crashed container
			if err = dm.dockerRestart(container.ID); err != nil {
				return "", err
			}
		}
	} else {
		if err = dm.dockerStart(container); err != nil {
			return "", err
		}
	}

	port, err = dm.getLambdaPort(img)
	if err != nil {
		return "", err
	}
	return port, nil
}

func (dm *DockerManager) dockerKill(id string) (err error) {
	// TODO(tyler): is there any advantage to trying to stop
	// before killing?  (i.e., use SIGTERM instead SIGKILL)
	opts := docker.KillContainerOptions{ID: id}
	if err = dm.client.KillContainer(opts); err != nil {
		log.Printf("failed to kill container with error %v\n", err)
		return dm.dockerError(id, err)
	}
	return nil
}

func (dm *DockerManager) dockerRestart(img string) (err error) {
	// Restart container after (0) seconds
	if err = dm.client.RestartContainer(img, 0); err != nil {
		log.Printf("failed to restart container with error %v\n", err)
		return dm.dockerError(img, err)
	}
	return nil
}

func (dm *DockerManager) dockerPause(img string) (err error) {
	dm.pauseTimer.Start()
	if err = dm.client.PauseContainer(img); err != nil {
		log.Printf("failed to pause container with error %v\n", err)
		return dm.dockerError(img, err)
	}
	dm.pauseTimer.Stop()

	return nil
}

func (dm *DockerManager) dockerUnpause(cid string) (err error) {
	dm.unpauseTimer.Start()
	if err = dm.client.UnpauseContainer(cid); err != nil {
		log.Printf("failed to unpause container %s with err %v\n", cid, err)
		return dm.dockerError(cid, err)
	}
	dm.unpauseTimer.Stop()

	return nil
}

func (dm *DockerManager) dockerPull(img string) error {
	dm.pullTimer.Start()
	err := dm.client.PullImage(
		docker.PullImageOptions{
			Repository: dm.registryName + "/" + img,
			Registry:   dm.registryName,
			Tag:        "latest",
		},
		docker.AuthConfiguration{},
	)
	dm.pullTimer.Stop()

	if err != nil {
		return fmt.Errorf("failed to pull '%v' from %v registry\n", img, dm.registryName)
	}

	err = dm.client.TagImage(
		dm.registryName+"/"+img,
		docker.TagImageOptions{Repo: img, Force: true})
	if err != nil {
		log.Printf("failed to re-tag container: %v\n", err)
		return fmt.Errorf("failed to re-tag container: %v\n", err)
	}

	return nil
}

// Left public for handler tests. Consider refactor
func (dm *DockerManager) DockerImageExists(img_name string) (bool, error) {
	_, err := dm.client.InspectImage(img_name)
	if err == docker.ErrNoSuchImage {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (dm *DockerManager) dockerContainerExists(cname string) (bool, error) {
	_, err := dm.client.InspectContainer(cname)
	if err != nil {
		switch err.(type) {
		default:
			return false, err
		case *docker.NoSuchContainer:
			return false, nil
		}
	}
	return true, nil
}

func (dm *DockerManager) dockerStart(container *docker.Container) (err error) {
	dm.startTimer.Start()
	if err = dm.client.StartContainer(container.ID, container.HostConfig); err != nil {
		log.Printf("failed to start container with err %v\n", err)
		return dm.dockerError(container.ID, err)
	}
	dm.startTimer.Stop()

	return nil
}

func (dm *DockerManager) dockerCreate(img string, args []string) (*docker.Container, error) {
	// Create a new container with img and args
	// Specifically give container name of img, so we can lookup later

	// A note on ports
	// lambdas ALWAYS use port 8080 internally, they are given a free random port externally
	// the client will later lookup the host port by finding which host port,
	// for a specific container is bound to 8080
	//
	// Using port 0 will force the OS to choose a free port for us.
	dm.createTimer.Start()
	port := 0
	portStr := strconv.Itoa(port)
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: portStr}}}
	container, err := dm.client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Cmd:          args,
				AttachStdout: true,
				AttachStderr: true,
				Image:        img,
				ExposedPorts: internalAppPort,
			},
			HostConfig: &docker.HostConfig{
				PortBindings:    portBindings,
				PublishAllPorts: true,
			},
			Name: img,
		},
	)
	dm.createTimer.Stop()

	if err != nil {
		// commented because at large scale, this isnt always an error, and therefor shouldnt polute logs
		// log.Printf("container %s failed to create with err: %v\n", img, err)
		return nil, dm.dockerError(img, err)
	}

	return container, nil
}

func (dm *DockerManager) dockerInspect(cid string) (container *docker.Container, err error) {
	dm.inspectTimer.Start()
	container, err = dm.client.InspectContainer(cid)
	if err != nil {
		log.Printf("failed to inspect %s with err %v\n", cid, err)
		return nil, dm.dockerError(cid, err)
	}
	dm.inspectTimer.Stop()

	return container, nil
}

func (dm *DockerManager) dockerRemove(container *docker.Container) (err error) {
	if err = dm.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: container.ID,
	}); err != nil {
		log.Printf("failed to rm container with err %v", err)
		return dm.dockerError(container.ID, err)
	}

	return nil
}

// Returned as "port"
func (dm *DockerManager) getLambdaPort(cid string) (port string, err error) {
	container, err := dm.dockerInspect(cid)
	if err != nil {
		return "", dm.dockerError(cid, err)
	}

	container_port := docker.Port("8080/tcp")
	ports := container.NetworkSettings.Ports[container_port]
	if len(ports) == 0 {
		err := fmt.Errorf("could not lookup host port for %v", container_port)
		return "", dm.dockerError(cid, err)
	} else if len(ports) > 1 {
		err := fmt.Errorf("multiple host port mapping to %v", container_port)
		return "", dm.dockerError(cid, err)
	}
	port = ports[0].HostPort

	// on unix systems, port is given as "unix:port", this removes the prefix
	if strings.HasPrefix(port, "unix") {
		port = strings.Split(port, ":")[1]
	}
	return port, nil
}

func (dm *DockerManager) Dump() {
	opts := docker.ListContainersOptions{All: true}
	containers, err := dm.client.ListContainers(opts)
	if err != nil {
		log.Fatal("Could not get container list")
	}
	log.Printf("=====================================\n")
	for idx, info := range containers {
		container, err := dm.dockerInspect(info.ID)
		if err != nil {
			log.Fatal("Could not get container")
		}

		log.Printf("CONTAINER %d: %v, %v, %v\n", idx,
			info.Image,
			container.ID[:8],
			container.State.String())
	}
	log.Printf("=====================================\n")
	log.Println()
	log.Printf("====== Docker Operation Stats =======\n")
	log.Printf("\tcreate: \t%fms\n", dm.createTimer.AverageMs())
	log.Printf("\tinspect: \t%fms\n", dm.inspectTimer.AverageMs())
	log.Printf("\tlogs: \t%fms\n", dm.logTimer.AverageMs())
	log.Printf("\tpause: \t\t%fms\n", dm.pauseTimer.AverageMs())
	log.Printf("\tpull: \t\t%fms\n", dm.pullTimer.AverageMs())
	log.Printf("\tremove: \t%fms\n", dm.removeTimer.AverageMs())
	log.Printf("\trestart: \t%fms\n", dm.restartTimer.AverageMs())
	log.Printf("\trestart: \t%fms\n", dm.restartTimer.AverageMs())
	log.Printf("\tunpause: \t%fms\n", dm.unpauseTimer.AverageMs())
	log.Printf("=====================================\n")
}

func (dm *DockerManager) Client() *docker.Client {
	return dm.client
}

func (dm *DockerManager) initTimers() {
	dm.createTimer = turnip.NewTurnip()
	dm.inspectTimer = turnip.NewTurnip()
	dm.pauseTimer = turnip.NewTurnip()
	dm.pullTimer = turnip.NewTurnip()
	dm.removeTimer = turnip.NewTurnip()
	dm.restartTimer = turnip.NewTurnip()
	dm.startTimer = turnip.NewTurnip()
	dm.unpauseTimer = turnip.NewTurnip()
	dm.logTimer = turnip.NewTurnip()
}

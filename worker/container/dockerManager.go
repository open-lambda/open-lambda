package container

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/phonyphonecall/turnip"
	"github.com/tylerharter/open-lambda/worker/handler/state"
)

type DockerManager struct {
	client *docker.Client

	registryName string

	// timers
	createTimer  *turnip.Turnip
	pauseTimer   *turnip.Turnip
	unpauseTimer *turnip.Turnip
	pullTimer    *turnip.Turnip
	restartTimer *turnip.Turnip
	inspectTimer *turnip.Turnip
	startTimer   *turnip.Turnip
	removeTimer  *turnip.Turnip
}

func NewDockerManager(host string, port string) (manager *DockerManager) {
	manager = new(DockerManager)

	// NOTE: This requires that users have pre-configured the environement a docker daemon
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		manager.client = c
	}

	manager.registryName = fmt.Sprintf("%s:%s", host, port)
	manager.initTimers()
	return manager
}

func (cm *DockerManager) PullAndCreate(img string, args []string) (container *docker.Container, err error) {
	if container, err = cm.DockerCreate(img, args); err != nil {
		// if the container already exists, don't pull, let client decide how to handle
		if err == docker.ErrContainerAlreadyExists {
			return nil, err
		}

		if err = cm.DockerPull(img); err != nil {
			log.Printf("img pull failed with: %v\n", err)
			return nil, err
		} else {
			container, err = cm.DockerCreate(img, args)
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
func (cm *DockerManager) DockerMakeReady(img string) (port string, err error) {
	// TODO: decide on one default lambda entry path
	container, err := cm.PullAndCreate(img, []string{})
	if err != nil {
		if err != docker.ErrContainerAlreadyExists {
			// Unhandled error
			return "", err
		}

		// make sure container is up
		cid := img
		container, err = cm.DockerInspect(cid)
		if err != nil {
			return "", err
		}
		if container.State.Paused {
			// unpause
			if err = cm.DockerUnpause(container.ID); err != nil {
				return "", err
			}
		} else if !container.State.Running {
			// restart a stopped/crashed container
			if err = cm.DockerRestart(container.ID); err != nil {
				return "", err
			}
		}
	} else {
		if err = cm.dockerStart(container); err != nil {
			return "", err
		}
	}

	port, err = cm.getLambdaPort(img)
	if err != nil {
		return "", err
	}
	return port, nil
}

func (cm *DockerManager) DockerKill(img string) (err error) {
	// TODO(tyler): is there any advantage to trying to stop
	// before killing?  (i.e., use SIGTERM instead SIGKILL)
	opts := docker.KillContainerOptions{ID: img}
	if err = cm.client.KillContainer(opts); err != nil {
		log.Printf("failed to kill container with error %v\n", err)
		return err
	}
	return nil
}

func (cm *DockerManager) DockerRestart(img string) (err error) {
	// Restart container after (0) seconds
	if err = cm.client.RestartContainer(img, 0); err != nil {
		log.Printf("failed to restart container with error %v\n", err)
		return err
	}
	return nil
}

func (cm *DockerManager) DockerPause(img string) (err error) {
	cm.pauseTimer.Start()
	if err = cm.client.PauseContainer(img); err != nil {
		log.Printf("failed to pause container with error %v\n", err)
		return err
	}
	cm.pauseTimer.Stop()

	return nil
}

func (cm *DockerManager) DockerUnpause(cid string) (err error) {
	cm.unpauseTimer.Start()
	if err = cm.client.UnpauseContainer(cid); err != nil {
		log.Printf("failed to unpause container %s with err %v\n", cid, err)
		return err
	}
	cm.unpauseTimer.Stop()

	return nil
}

func (cm *DockerManager) DockerPull(img string) error {
	cm.pullTimer.Start()
	err := cm.client.PullImage(
		docker.PullImageOptions{
			Repository: cm.registryName + "/" + img,
			Registry:   cm.registryName,
			Tag:        "latest",
		},
		docker.AuthConfiguration{},
	)
	cm.pullTimer.Stop()

	if err != nil {
		log.Printf("failed to pull container: %v\n", err)
		return err
	}

	err = cm.client.TagImage(
		cm.registryName+"/"+img,
		docker.TagImageOptions{Repo: img, Force: true})
	if err != nil {
		log.Printf("failed to re-tag container: %v\n", err)
		return err
	}

	return nil
}

// Combines a docker create with a docker start
func (cm *DockerManager) DockerRun(img string, args []string, waitAndRemove bool) (err error) {
	c, err := cm.DockerCreate(img, args)
	if err != nil {
		return err
	}
	err = cm.dockerStart(c)
	if err != nil {
		return err
	}

	if waitAndRemove {
		// img == cid in our create container
		_, err = cm.client.WaitContainer(img)
		if err != nil {
			log.Printf("failed to wait on container %s with err %v\n", img, err)
			return err
		}
		err = cm.dockerRemove(c)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cm *DockerManager) DockerImageExists(img_name string) (bool, error) {
	_, err := cm.client.InspectImage(img_name)
	if err == docker.ErrNoSuchImage {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (cm *DockerManager) DockerContainerExists(cname string) (bool, error) {
	_, err := cm.client.InspectContainer(cname)
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

func (cm *DockerManager) dockerStart(container *docker.Container) (err error) {
	cm.startTimer.Start()
	if err = cm.client.StartContainer(container.ID, container.HostConfig); err != nil {
		log.Printf("failed to start container with err %v\n", err)
		return err
	}
	cm.startTimer.Stop()

	return nil
}

func (cm *DockerManager) DockerCreate(img string, args []string) (*docker.Container, error) {
	// Create a new container with img and args
	// Specifically give container name of img, so we can lookup later

	// A note on ports
	// lambdas ALWAYS use port 8080 internally, they are given a free random port externally
	// the client will later lookup the host port by finding which host port,
	// for a specific container is bound to 8080
	//
	// Using port 0 will force the OS to choose a free port for us.
	cm.createTimer.Start()
	port := 0
	portStr := strconv.Itoa(port)
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: portStr}}}
	container, err := cm.client.CreateContainer(
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
	cm.createTimer.Stop()

	if err != nil {
		// commented because at large scale, this isnt always an error, and therefor shouldnt polute logs
		// log.Printf("container %s failed to create with err: %v\n", img, err)
		return nil, err
	}

	return container, nil
}

func (cm *DockerManager) DockerInspect(cid string) (container *docker.Container, err error) {
	cm.inspectTimer.Start()
	container, err = cm.client.InspectContainer(cid)
	if err != nil {
		log.Printf("failed to inspect %s with err %v\n", cid, err)
		return nil, err
	}
	cm.inspectTimer.Stop()

	return container, nil
}

func (cm *DockerManager) dockerRemove(container *docker.Container) (err error) {
	if err = cm.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: container.ID,
	}); err != nil {
		log.Printf("failed to rm container with err %v", err)
		return err
	}

	return nil
}

// Returned as "port"
func (cm *DockerManager) getLambdaPort(cid string) (port string, err error) {
	container, err := cm.DockerInspect(cid)
	if err != nil {
		return "", err
	}

	// TODO: Will we ever need to look at other ip's than the first?
	port = container.NetworkSettings.Ports["8080/tcp"][0].HostPort

	// on unix systems, port is given as "unix:port", this removes the prefix
	if strings.HasPrefix(port, "unix") {
		port = strings.Split(port, ":")[1]
	}
	return port, nil
}

func (cm *DockerManager) Dump() {
	opts := docker.ListContainersOptions{All: true}
	containers, err := cm.client.ListContainers(opts)
	if err != nil {
		log.Fatal("Could not get container list")
	}
	log.Printf("=====================================\n")
	for idx, info := range containers {
		container, err := cm.DockerInspect(info.ID)
		if err != nil {
			log.Fatal("Could get container")
		}

		log.Printf("CONTAINER %d: %v, %v, %v\n", idx,
			info.Image,
			container.ID[:8],
			container.State.String())
	}
	log.Printf("=====================================\n")
	log.Println()
	log.Printf("====== Docker Operation Stats =======\n")
	log.Printf("\tcreate: \t%fms\n", cm.createTimer.AverageMs())
	log.Printf("\tinspect: \t%fms\n", cm.inspectTimer.AverageMs())
	log.Printf("\tpause: \t\t%fms\n", cm.pauseTimer.AverageMs())
	log.Printf("\tpull: \t\t%fms\n", cm.pullTimer.AverageMs())
	log.Printf("\tremove: \t%fms\n", cm.removeTimer.AverageMs())
	log.Printf("\trestart: \t%fms\n", cm.restartTimer.AverageMs())
	log.Printf("\trestart: \t%fms\n", cm.restartTimer.AverageMs())
	log.Printf("\tunpause: \t%fms\n", cm.unpauseTimer.AverageMs())
	log.Printf("=====================================\n")
}

func (cm *DockerManager) Client() *docker.Client {
	return cm.client
}

func (cm *DockerManager) initTimers() {
	cm.createTimer = turnip.NewTurnip()
	cm.inspectTimer = turnip.NewTurnip()
	cm.pauseTimer = turnip.NewTurnip()
	cm.pullTimer = turnip.NewTurnip()
	cm.removeTimer = turnip.NewTurnip()
	cm.restartTimer = turnip.NewTurnip()
	cm.startTimer = turnip.NewTurnip()
	cm.unpauseTimer = turnip.NewTurnip()
}

// Runs any preperation to get the container ready to run
func (cm *DockerManager) MakeReady(name string) (info ContainerInfo, err error) {
	// make sure image is pulled
	imgExists, err := cm.DockerImageExists(name)
	if err != nil {
		return info, err
	}
	if !imgExists {
		if err := cm.DockerPull(name); err != nil {
			return info, err
		}
	}

	// make sure container is created
	contExists, err := cm.DockerContainerExists(name)
	if err != nil {
		return info, err
	}
	if !contExists {
		if _, err := cm.DockerCreate(name, []string{}); err != nil {
			return info, err
		}
	}

	return cm.GetInfo(name)
}

// Returns the current state of the container
// If a container has never been started, the port will be -1
func (cm *DockerManager) GetInfo(name string) (info ContainerInfo, err error) {
	container, err := cm.DockerInspect(name)
	if err != nil {
		return info, err
	}

	// TODO: can the State enum be both paused and running?
	var hState state.HandlerState
	if container.State.Running {
		if container.State.Paused {
			hState = state.Paused
		} else {
			hState = state.Running
		}
	} else {
		hState = state.Stopped
	}

	// If the container has never been started, it will have no port
	port := "-1"
	if hState != state.Stopped {
		port, err = cm.getLambdaPort(name)
		if err != nil {
			return info, err
		}
	}

	info.State = hState
	info.Port = port

	return info, nil
}

// Starts a given container
func (cm *DockerManager) Start(name string) error {
	c, err := cm.DockerInspect(name)
	if err != nil {
		return err
	}

	return cm.dockerStart(c)
}

// Pauses a given container
func (cm *DockerManager) Pause(name string) error {
	return cm.DockerPause(name)
}

// Unpauses a given container
func (cm *DockerManager) Unpause(name string) error {
	return cm.DockerUnpause(name)
}

// Stops a given container
func (cm *DockerManager) Stop(name string) error {
	return cm.DockerKill(name)
}

// Frees all resources associated with a given lambda
// Will stop if needed
func (cm *DockerManager) Remove(name string) error {
	container, err := cm.DockerInspect(name)
	if err != nil {
		return err
	}

	return cm.dockerRemove(container)
}

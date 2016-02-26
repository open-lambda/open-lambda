package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	docker "github.com/fsouza/go-dockerclient"
)

type ContainerManager struct {
	client       *docker.Client
	registryName string
}

func NewContainerManager(host string, port string) (manager *ContainerManager) {
	manager = new(ContainerManager)

	// NOTE: This requires that users haev pre-configured the environement a docker daemon
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		manager.client = c
	}

	manager.registryName = fmt.Sprintf("%s:%s", host, port)
	return manager
}

func (cm *ContainerManager) createContainer(img string, args []string) (*docker.Container, error) {
	// Create a new container with img and args
	// Specifically give container name of img, so we can lookup later

	// A note on ports
	// lambdas ALWAYS use port 8080 internally, they are given a random port externally
	// the client will later lookup the host port by finding which host port,
	// for a specific container is bound to 8080
	port, err := getFreePort()
	if err != nil {
		log.Printf("failed to get free port with err %v\n", err)
		return nil, err
	}

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
	if err != nil {
		// commented because at large scale, this isnt always an error, and therefor shouldnt polute logs
		// log.Printf("container %s failed to create with err: %v\n", img, err)
		return nil, err
	}

	return container, nil
}

func (cm *ContainerManager) pullAndCreate(img string, args []string) (container *docker.Container, err error) {
	if container, err = cm.createContainer(img, args); err != nil {
		// if the container already exists, don't pull, let client decide how to handle
		if strings.Contains(err.Error(), "already exists") {
			return nil, err
		}

		if err = cm.DockerPull(img); err != nil {
			log.Printf("img pull failed with: %v\n", err)
			return nil, err
		} else {
			container, err = cm.createContainer(img, args)
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
func (cm *ContainerManager) DockerMakeReady(img string) (port string, err error) {
	// TODO: decide on one default lambda entry path
	container, err := cm.pullAndCreate(img, []string{"/go/bin/app"})
	if err != nil {
		if !strings.Contains(err.Error(), "container already exists") {
			// Unhandled error
			return "", err
		}

		// make sure container is up
		cid := img
		container, err = cm.dockerInspect(cid)
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

func (cm *ContainerManager) DockerRestart(img string) (err error) {
	// Restart container after (0) seconds
	if err = cm.client.RestartContainer(img, 0); err != nil {
		log.Printf("failed to pause container with error %v\n", err)
		return err
	}
	return nil
}

func (cm *ContainerManager) DockerPause(img string) (err error) {
	if err = cm.client.PauseContainer(img); err != nil {
		log.Printf("failed to pause container with error %v\n", err)
		return err
	}
	return nil
}

func (cm *ContainerManager) DockerUnpause(cid string) (err error) {
	if err = cm.client.UnpauseContainer(cid); err != nil {
		log.Printf("failed to unpause container %s with err %v\n", cid, err)
		return err
	}
	return nil
}

func (cm *ContainerManager) DockerPull(img string) error {
	err := cm.client.PullImage(
		docker.PullImageOptions{
			Repository: img,
			Registry:   cm.registryName,
		},
		docker.AuthConfiguration{},
	)

	if err != nil {
		log.Printf("failed to pull container: %v\n", err)
		return err
	}
	return nil
}

func (cm *ContainerManager) dockerInspect(cid string) (container *docker.Container, err error) {
	container, err = cm.client.InspectContainer(cid)
	if err != nil {
		log.Printf("failed to inspect %s with err %v\n", cid, err)
		return nil, err
	}
	return container, nil
}

func (cm *ContainerManager) dockerStart(container *docker.Container) (err error) {
	if err = cm.client.StartContainer(container.ID, container.HostConfig); err != nil {
		log.Printf("failed to start container with err %v\n", err)
		return err
	}
	return nil
}

func (cm *ContainerManager) dockerRemove(container *docker.Container) (err error) {
	if err = cm.client.RemoveContainer(docker.RemoveContainerOptions{
		ID: container.ID,
	}); err != nil {
		log.Println("failed to rm container with err %v", err)
		return err
	}

	return nil
}

// Returned as "port"
func (cm *ContainerManager) getLambdaPort(cid string) (port string, err error) {
	container, err := cm.dockerInspect(cid)
	if err != nil {
		return "", err
	}

	// TODO: Will we ever need to look at other ip's than the first?
	port = container.HostConfig.PortBindings["8080/tcp"][0].HostPort

	// on unix systems, port is given as "unix:port", this removes the prefix
	if strings.HasPrefix(port, "unix") {
		port = strings.Split(port, ":")[1]
	}
	return port, nil
}

// TODO: This is NOT thread safe
// 		  Someone can steal the port between when we return,
//		  And when it is used.
func getFreePort() (port int, err error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		log.Println("os failed to give us good port with err %v", err)
		return -1, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Println("failed to listen, someone stole our port! %v", err)
		return -1, err
	}
	defer l.Close()
	port = l.Addr().(*net.TCPAddr).Port
	return port, nil
}

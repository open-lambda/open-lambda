package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

var client *docker.Client

var registryName string

func init() {
	// TODO: This requires that users haev pre-configured the environement a docker daemon
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		client = c
	}
	log.Println("docker client initialized")
}

func SetRegistry(host string, port string) {
	registryName = fmt.Sprintf("%s:%s", host, port)
}

func pullContainer(img string) error {
	err := client.PullImage(
		docker.PullImageOptions{
			Repository: img,
			Registry:   registryName,
		},
		docker.AuthConfiguration{})
	log.Printf("pull of %s complete\n", img)

	if err != nil {
		log.Printf("failed to pull container: %v\n", err)
	}
	return err
}

func createContainer(img string, args []string) (*docker.Container, error) {
	// Create a new container with img and args
	// Specifically give container name of img, so we can lookup later

	// A note on ports
	// lambdas ALWAYS use port 8080 internally
	// they are given a random port externally
	// the client will later lookup the host port by
	// finding which host port, for a specific
	// container is bound to 8080
	// TODO: randomize ports
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "8080"}}}
	container, err := client.CreateContainer(
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
		log.Printf("container %s failed to create with err: %v\n", img, err)
	}
	return container, err
}

func removeContainer(container *docker.Container) error {
	beforeRemoval := time.Now()
	// remove container
	err := client.RemoveContainer(docker.RemoveContainerOptions{
		ID: container.ID,
	})
	afterRemoval := time.Now()
	log.Printf("container removal took %v\n", afterRemoval.Sub(beforeRemoval))
	if err != nil {
		log.Println("failed to rm container")
		return err
	}

	return nil
}

func pullAndCreate(img string, args []string) (container *docker.Container, err error) {
	container, err = createContainer(img, args)
	if err != nil {
		// if the container already exists, don't pull
		// let client decide how to handle
		if strings.Contains(err.Error(), "already exists") {
			return nil, err
		}

		err = pullContainer(img)
		if err != nil {
			log.Printf("container creation failed with: %v\n", err)
			log.Printf("img pull failed with: %v\n", err)
			return nil, err
		} else {
			container, err = createContainer(img, args)
			if err != nil {
				log.Printf("failed to create container %s after good pull, with error: %v\n", img, err)
				return nil, err
			}
		}
	}
	return container, nil
}

func DockerRunImg(img string, args []string) (stdout string, stderr string, err error) {
	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	// create container
	container, err := pullAndCreate(img, args)
	if err != nil {
		return "", "", err
	}

	// Then run it
	err = client.StartContainer(container.ID, container.HostConfig)
	if err != nil {
		log.Println("failed to start container")
		return "", "", err
	}

	// Wait for container to finish
	code, err := client.WaitContainer(container.ID)
	if err != nil {
		log.Println("wait failed: ", code, err)
		return "", "", err
	}

	// Fill buffers with logs
	err = client.AttachToContainer(docker.AttachToContainerOptions{
		Container:    container.ID,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
		Logs:         true,
		Stdout:       true,
		Stderr:       true,
	})

	if err != nil {
		log.Fatal("failed to attach to container\n", err)
	}

	err = removeContainer(container)

	return outBuf.String(), errBuf.String(), nil
}

func getContainerAddress(container *docker.Container) (address string, err error) {
	// This gives us: "https://<daemon_ip>:<docker_port>"
	host := client.Endpoint()
	// We want: "<daemon_ip>:<lambda_port>"

	// trim bad protocol
	host = strings.TrimPrefix(host, "https://")
	// trim old port, keeping ":"
	idx := strings.Index(host, ":")
	host = host[:idx+1]

	var buf bytes.Buffer
	buf.WriteString(host)
	// add new port
	for p1, _ := range container.Config.ExposedPorts {
		buf.WriteString(p1.Port())
		break
	}
	return buf.String(), nil
}

// returns the running container host in "<host>:<port>" form
func DockerMakeReady(img string) (containerHost string, err error) {
	// TODO: decide on one default lambda entry path
	container, err := pullAndCreate(img, []string{"/go/bin/app"})
	if err != nil {
		if strings.Contains(err.Error(), "container already exists") {
			// make sure container is up
			// pullAndCraete always sets cId to img
			containerId := img
			container, err = client.InspectContainer(containerId)
			if err != nil {
				log.Printf("failed to inspect %s with err %v\n", containerId, err)
				return "", err
			}
			if container.State.Paused {
				// unpause
				err = client.UnpauseContainer(container.ID)
				if err != nil {
					log.Printf("failed to unpause container %s with err %v\n", container.ID, err)
					return "", err
				}

				return getContainerAddress(container)
			} else if container.State.Running {
				// Good to go
				return getContainerAddress(container)
			}
		} else {
			return "", err
		}
	}

	err = client.StartContainer(container.ID, container.HostConfig)
	if err != nil {
		log.Printf("failed to start container with err %v\n", err)
		return "", err
	}

	return getContainerAddress(container)
}

func DockerPause(img string) (err error) {
	if err = client.PauseContainer(img); err != nil {
		log.Printf("failed to pause container with error %v\n", err)
		return err
	}
	return nil
}

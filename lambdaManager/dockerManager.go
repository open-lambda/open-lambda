package main

import (
	"bytes"
	"fmt"
	"log"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

var client *docker.Client

var registryName string

func initClient() {

	// TODO: This requires that users haev pre-configured the environement to swarm manager
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		client = c
	}
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
	beforeContainerTime := time.Now()

	// Create a new container with img and args
	container, err := client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Cmd:          args,
				AttachStdout: true,
				AttachStderr: true,
				Image:        img,
			},
			HostConfig: &docker.HostConfig{},
		},
	)

	if err == nil {
		afterContainerTime := time.Now()
		log.Printf("container creation took %v\n",
			afterContainerTime.Sub(beforeContainerTime))
	} else {
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

func DockerRunImg(img string, args []string) (stdout string, stderr string, err error) {
	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	if client == nil {
		initClient()
	}

	var container *docker.Container
	container, err = createContainer(img, args)
	if err != nil {
		// assume failed because need to pull container
		err = pullContainer(img)
		if err != nil {
			log.Printf("container creation failed with: %v\n")
			log.Printf("img pull failed with: %v\n")
			return "", "", err
		} else {
			container, err = createContainer(img, args)
			if err != nil {
				log.Printf("failed to create container %s after good pull, with error: %v\n", img, err)
			}
		}
	}

	beforeRunTime := time.Now()
	// Then run it
	err = client.StartContainer(container.ID, container.HostConfig)
	if err != nil {
		log.Println("failed to start container")
		return "", "", err
	}
	afterRunStartTime := time.Now()
	log.Printf("run issuing took %v\n", afterRunStartTime.Sub(beforeRunTime))

	// Wait for container to finish
	code, err := client.WaitContainer(container.ID)
	if err != nil {
		log.Println("wait failed: ", code, err)
		return "", "", err
	}

	afterContainerExitTime := time.Now()
	log.Printf("container run took approx %v\n", afterContainerExitTime.Sub(beforeRunTime))

	beforeLogGetTime := time.Now()
	// Fill buffers with logs
	err = client.AttachToContainer(docker.AttachToContainerOptions{
		Container:    container.ID,
		OutputStream: &outBuf,
		ErrorStream:  &errBuf,
		Logs:         true,
		Stdout:       true,
		Stderr:       true,
	})
	afterLogGetTime := time.Now()
	log.Printf("log getting took approx %v\n", afterLogGetTime.Sub(beforeLogGetTime))
	if err != nil {
		log.Fatal("failed to attach to container\n", err)
	}

	err = removeContainer(container)

	return outBuf.String(), errBuf.String(), nil
}

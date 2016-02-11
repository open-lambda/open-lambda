package main

import (
	"bytes"
	"log"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

var client *docker.Client

func initClient() {
	// TODO: This requires that users haev pre-configured the environement to swarm manager
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		client = c
	}
}

func createContainer(img string, args []string) (*docker.Container, error) {
	beforeContainerTime := time.Now()
	// Create a new container, with correct args
	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Cmd:          args,
			AttachStdout: true,
			AttachStderr: true,
			Image:        img,
		},
		HostConfig: &docker.HostConfig{},
	})
	afterContainerTime := time.Now()
	log.Printf("container creation took %v\n", afterContainerTime.Sub(beforeContainerTime))

	// TODO: This case should attempt to pull img
	if err != nil {
		log.Println("failed to create container", err)
		return nil, err
	}

	return container, nil
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

	container, err := createContainer(img, args)
	if err != nil {
		return "", "", err
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

package dockerManager

import (
	"bytes"
	"log"

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

func RunImg(img string, args []string) (stdout string, stderr string, err error) {
	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)

	if client == nil {
		initClient()
	}

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
	// TODO: This case should attempt to pull container
	if err != nil {
		log.Println("failed to create container", err)
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

	// remove container
	err = client.RemoveContainer(docker.RemoveContainerOptions{
		ID: container.ID,
	})
	if err != nil {
		log.Println("failed to rm container")
		return "", "", nil
	}

	return outBuf.String(), errBuf.String(), nil
}

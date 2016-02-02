package dockerManager

import (
	"bytes"
	"log"

	docker "github.com/fsouza/go-dockerclient"
)

func RunImg(img string, args []string) (stdout string, stderr string, err error) {
	var (
		outBuf bytes.Buffer
		errBuf bytes.Buffer
	)
	// TODO: This requires that users haev pre-configured the environement to swarm manager
	if client, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		// Ensure we are actually connected...
		if _, err := client.Info(); err != nil {
			log.Println("failed to get info: %v", err)
		} else {
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
				log.Fatal(err)
			}

			// remove container
			err = client.RemoveContainer(docker.RemoveContainerOptions{
				ID: container.ID,
			})
			if err != nil {
				log.Println("failed to rm container")
				return "", "", nil
			}
		}
	}

	return outBuf.String(), errBuf.String(), nil
}

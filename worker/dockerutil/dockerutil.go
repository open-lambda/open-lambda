// package includes utility functions not provided by go-dockerclient
package dockerutil

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"

	docker "github.com/fsouza/go-dockerclient"
)

const (
	DOCKER_LABEL_CLUSTER = "ol.cluster" // cluster name
	DOCKER_LABEL_TYPE    = "ol.type"    // container type (sb, olstore, rethinkdb, etc)
	SANDBOX              = "sandbox"
	BASE_IMAGE           = "lambda"
	POOL                 = "pool"
	POOL_IMAGE           = "server-pool"
)

// ImageExists checks if an image of name exists.
func ImageExists(client *docker.Client, name string) (bool, error) {
	_, err := client.InspectImage(name)
	if err == docker.ErrNoSuchImage {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// Prints the ID and state of all containers. Only for debugging.
func Dump(client *docker.Client) {
	opts := docker.ListContainersOptions{All: true}
	containers, err := client.ListContainers(opts)
	if err != nil {
		log.Fatal("Could not get container list")
	}
	log.Printf("=====================================\n")
	for idx, info := range containers {
		container, err := client.InspectContainer(info.ID)
		if err != nil {
			log.Fatal("Could not get container")
		}

		log.Printf("CONTAINER %d: %v, %v, %v\n", idx,
			info.Image,
			container.ID[:8],
			container.State.String())
	}
}

// take a Docker image, and extract a flattened version to a local directory
func DumpDockerImage(client *docker.Client, image string, outdir string) error {
	if err := os.Mkdir(outdir, 0700); err != nil {
		return err
	}

	// we will pipe the output of "docker export" to "tar xf ..."
	tar := exec.Command("tar", "xf", "-", "--directory", outdir)
	writer, err := tar.StdinPipe()
	tar.Stdout = os.Stdout
	tar.Stderr = os.Stderr
	if err != nil {
		return err
	}

	// dump tar of base image async
	err_chan := make(chan error)
	go func() {
		err_chan <- func() error {
			defer writer.Close()

			cmd := []string{"sleep", "infinity"}

			container, err := client.CreateContainer(
				docker.CreateContainerOptions{
					Config: &docker.Config{
						Cmd:   cmd,
						Image: image,
					},
				},
			)
			if err != nil {
				return err
			}

			opts := docker.ExportContainerOptions{
				ID:           container.ID,
				OutputStream: writer,
			}
			if err := client.ExportContainer(opts); err != nil {
				return err
			}

			return client.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID})
		}()
	}()

	tar_err := tar.Run()
	export_err := <-err_chan

	// log both errors
	if export_err != nil {
		fmt.Printf("Docker export failed: %v\n", export_err.Error())
	}
	if tar_err != nil {
		fmt.Printf("Tar failed: %v\n", tar_err.Error())
	}

	// return one of the errors (if any)
	if export_err != nil {
		return export_err
	} else if tar_err != nil {
		return tar_err
	}

	// create mount points
	if err := os.Mkdir(path.Join(outdir, "handler"), 0700); err != nil {
		return err
	}

	if err := os.Mkdir(path.Join(outdir, "host"), 0700); err != nil {
		return err
	}

	return nil
}

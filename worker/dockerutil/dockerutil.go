// package includes utility functions not provided by go-dockerclient
package dockerutil

import (
	"log"

	docker "github.com/fsouza/go-dockerclient"
)

const (
	DOCKER_LABEL_CLUSTER = "ol.cluster" // cluster name
	DOCKER_LABEL_TYPE    = "ol.type"    // container type (sb, olstore, rethinkdb, etc)
	SANDBOX              = "sandbox"
	BASE_IMAGE           = "lambda"
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

/*

Defines common variables and functions to be shared
by managers which managing Docker containers.

*/

package manager

import (
	"fmt"
	"log"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/config"
)

const (
	DOCKER_LABEL_CLUSTER = "ol.cluster"
	DOCKER_LABEL_TYPE    = "ol.type"
	SANDBOX              = "sandbox"
	BASE_IMAGE           = "lambda"
)

type DockerManagerBase struct {
	opts    *config.Config
	dClient *docker.Client
	env     []string
}

func (manager *DockerManagerBase) init(opts *config.Config) {
	// NOTE: This requires a running docker daemon on the host
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		manager.dClient = c
	}
	manager.env = []string{fmt.Sprintf("ol.config=%s", opts.SandboxConfJson())}

	manager.opts = opts
}

func (dm *DockerManagerBase) docker_labels() map[string]string {
	labels := map[string]string{}
	labels[DOCKER_LABEL_CLUSTER] = dm.opts.Cluster_name
	labels[DOCKER_LABEL_TYPE] = SANDBOX
	return labels
}

func (dm *DockerManagerBase) client() *docker.Client {
	return dm.dClient
}

func (dm *DockerManagerBase) Dump() {
	opts := docker.ListContainersOptions{All: true}
	containers, err := dm.client().ListContainers(opts)
	if err != nil {
		log.Fatal("Could not get container list")
	}
	log.Printf("=====================================\n")
	for idx, info := range containers {
		container, err := dm.client().InspectContainer(info.ID)
		if err != nil {
			log.Fatal("Could not get container")
		}

		log.Printf("CONTAINER %d: %v, %v, %v\n", idx,
			info.Image,
			container.ID[:8],
			container.State.String())
	}
}

// Left public for handler tests. Consider refactor
func (dm *DockerManagerBase) DockerImageExists(img_name string) (bool, error) {
	_, err := dm.dClient.InspectImage(img_name)
	if err == docker.ErrNoSuchImage {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

package sbmanager

/*

Defines common variables and functions to be shared
by managers which managing Docker containers.

*/

import (
	"fmt"
	"log"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/config"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
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

func (dm *DockerManagerBase) init(opts *config.Config) {
	// NOTE: This requires a running docker daemon on the host
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		dm.dClient = c
	}
	dm.env = []string{fmt.Sprintf("ol.config=%s", opts.SandboxConfJson())}

	dm.opts = opts
}

func (dm *DockerManagerBase) create(name string, sandbox_dir string, image string, volumes []string) (sb.Sandbox, error) {
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{ //TODO: don't need these with sockets
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "0"}}}

	var cmd []string
	if dm.opts.Pool == "" {
		cmd = []string{"/server.py"}
	} else {
		cmd = []string{"/init"}
	}

	container, err := dm.client().CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:        image,
				AttachStdout: true, //TODO: why do we need these?
				AttachStderr: true,
				ExposedPorts: internalAppPort,
				Labels:       dm.docker_labels(),
				Env:          dm.env,
				Cmd:          cmd,
			},
			HostConfig: &docker.HostConfig{
				PortBindings:    portBindings,
				PublishAllPorts: true,
				Binds:           volumes,
			},
		},
	)

	if err != nil {
		return nil, err
	}

	sandbox := sb.NewDockerSandbox(name, sandbox_dir, container, dm.client(), dm.opts)

	return sandbox, nil
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

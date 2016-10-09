package sandbox

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/open-lambda/open-lambda/worker/config"

	docker "github.com/fsouza/go-dockerclient"
)

type LocalManager struct {
	opts        *config.Config
	handler_dir string
	dClient     *docker.Client
}

func NewLocalManager(opts *config.Config) (manager *LocalManager) {
	manager = new(LocalManager)

	// NOTE: This requires that users have pre-configured the environement a docker daemon
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		manager.dClient = c
	}

	manager.opts = opts
	manager.handler_dir = opts.Reg_dir

	return manager
}

func (lm *LocalManager) Create(name string) (Sandbox, error) {
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "0"}}}
	labels := map[string]string{"openlambda.cluster": lm.opts.Cluster_name}

	handler := filepath.Join(lm.handler_dir, name)
	volumes := []string{fmt.Sprintf("%s:%s", handler, "/handler/")}

	container, err := lm.dClient.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:        "eoakes/lambda:latest",
				AttachStdout: true,
				AttachStderr: true,
				ExposedPorts: internalAppPort,
				Labels:       labels,
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

	sandbox := &DockerSandbox{name: name, container: container, mgr: lm}
	return sandbox, nil
}

func (lm *LocalManager) Pull(name string) error {
	path := filepath.Join(lm.handler_dir, name)
	_, err := os.Stat(path)

	return err

}

func (lm *LocalManager) Dump() {
	opts := docker.ListContainersOptions{All: true}
	containers, err := lm.dClient.ListContainers(opts)
	if err != nil {
		log.Fatal("Could not get container list")
	}
	log.Printf("=====================================\n")
	for idx, info := range containers {
		container, err := lm.dClient.InspectContainer(info.ID)
		if err != nil {
			log.Fatal("Could not get container")
		}

		log.Printf("CONTAINER %d: %v, %v, %v\n", idx,
			info.Image,
			container.ID[:8],
			container.State.String())
	}
}

func (lm *LocalManager) client() *docker.Client {
	return lm.dClient
}

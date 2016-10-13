package sandbox

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/open-lambda/open-lambda/worker/config"

	docker "github.com/fsouza/go-dockerclient"
)

type LocalManager struct {
	DockerManagerBase
	handler_dir string
}

func NewLocalManager(opts *config.Config) (manager *LocalManager) {
	manager = new(LocalManager)
	manager.DockerManagerBase.init(opts)
	manager.handler_dir = opts.Reg_dir
	return manager
}

func (lm *LocalManager) Create(name string) (Sandbox, error) {
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "0"}}}

	handler := filepath.Join(lm.handler_dir, name)
	volumes := []string{fmt.Sprintf("%s:%s", handler, "/handler/")}

	container, err := lm.client().CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:        "eoakes/lambda:latest",
				AttachStdout: true,
				AttachStderr: true,
				ExposedPorts: internalAppPort,
				Labels:       lm.docker_labels(),
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

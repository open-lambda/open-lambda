package sandbox

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	docker "github.com/fsouza/go-dockerclient"
	r "github.com/open-lambda/open-lambda/registry/src"
	"github.com/open-lambda/open-lambda/worker/config"
)

type RegistryManager struct {
	DockerManagerBase
	reg         *r.PullClient
	handler_dir string
}

func NewRegistryManager(opts *config.Config) (manager *RegistryManager) {
	manager = new(RegistryManager)
	manager.DockerManagerBase.init(opts)
	manager.reg = r.InitPullClient(opts.Reg_cluster)
	manager.handler_dir = "/var/tmp/olhandlers/"
	if err := os.Mkdir(manager.handler_dir, os.ModeDir); err != nil {
		err = os.RemoveAll(manager.handler_dir)
		if err != nil {
			log.Fatal("failed to remove old handler directory: ", err)
		}
		err = os.Mkdir(manager.handler_dir, os.ModeDir)
		if err != nil {
			log.Fatal("failed to create handler directory: ", err)
		}
	}

	return manager
}

func (rm *RegistryManager) Create(name string) (Sandbox, error) {
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "0"}}}

	handler := filepath.Join(rm.handler_dir, name)
	volumes := []string{fmt.Sprintf("%s:%s", handler, "/handler/")}

	container, err := rm.client().CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:        "eoakes/lambda:latest",
				AttachStdout: true,
				AttachStderr: true,
				ExposedPorts: internalAppPort,
				Labels:       rm.docker_labels(),
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

	sandbox := &DockerSandbox{name: name, container: container, mgr: rm}
	return sandbox, nil
}

func (rm *RegistryManager) Pull(name string) error {
	dir := filepath.Join(rm.handler_dir, name)
	if err := os.Mkdir(dir, os.ModeDir); err != nil {
		return err
	}

	handler := rm.reg.Pull(name)
	r := bytes.NewReader(handler)

	cmd := exec.Command("tar", "-xvzf", "-", "--directory", dir)
	cmd.Stdin = r
	return cmd.Run()

}

func (rm *RegistryManager) HandlerPresent(name string) (bool, error) {
	dir := filepath.Join(rm.handler_dir, name)
	_, err := os.Stat(dir)
	if err != nil {
		return false, nil
	}

	return true, nil
}

package manager

/*

Manages lambdas using the OpenLambda registry (built on RethinkDB).

Creates lambda containers using the generic base image defined in
dockerManagerBase.go (BASE_IMAGE).

Handler code is mapped into the container by attaching a directory
(<handler_dir>/<lambda_name>) when the container is started.

*/

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
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type RegistryManager struct {
	DockerManagerBase
	pullclient  *r.PullClient
	handler_dir string
}

func NewRegistryManager(opts *config.Config) (manager *RegistryManager, err error) {
	manager = new(RegistryManager)
	manager.DockerManagerBase.init(opts)
	manager.pullclient = r.InitPullClient(opts.Reg_cluster, r.DATABASE, r.TABLE)
	manager.handler_dir = "/var/tmp/olhandlers/"

	// Initialize a directory for the handler code. This directory is
	// mapped into the lambda container in RegistryManager.Create
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

	// Check that we have the base image for the lambda containers
	exists, err := manager.DockerImageExists(BASE_IMAGE)
	if err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("Docker image %s does not exist", BASE_IMAGE)
	}

	return manager, nil
}

func (rm *RegistryManager) Create(name string, sandbox_dir string) (sb.Sandbox, error) {
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "0"}}}

	handler := filepath.Join(rm.handler_dir, name)
	volumes := []string{
		fmt.Sprintf("%s:%s", handler, "/handler/"),
		fmt.Sprintf("%s:%s", sandbox_dir, "/host/")}

	container, err := rm.client().CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:        BASE_IMAGE,
				AttachStdout: true,
				AttachStderr: true,
				ExposedPorts: internalAppPort,
				Labels:       rm.docker_labels(),
				Env:          rm.env,
			},
			HostConfig: &docker.HostConfig{
				PortBindings:    portBindings,
				PublishAllPorts: true,
				Binds:           volumes, // attach handler code
			},
		},
	)

	if err != nil {
		return nil, err
	}

	nspid, err := rm.getNsPid(container)
	if err != nil {
		return nil, err
	}

	sandbox := sb.NewDockerSandbox(name, sandbox_dir, nspid, container, rm.client())

	return sandbox, nil
}

func (rm *RegistryManager) Pull(name string) error {
	dir := filepath.Join(rm.handler_dir, name)
	if err := os.Mkdir(dir, os.ModeDir); err != nil {
		return err
	}

	pfiles := rm.pullclient.Pull(name)
	handler := pfiles[r.HANDLER].([]byte)
	r := bytes.NewReader(handler)

	// TODO: try to uncompress without execing - faster?
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

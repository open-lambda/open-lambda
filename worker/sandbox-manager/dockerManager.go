package manager

/*

Manages lambdas using the Docker registry.

Each lambda endpoint must have an associated container image
in the registry, named with its ID.

*/

import (
	"fmt"
	"log"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/config"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type DockerManager struct {
	DockerManagerBase
	registryName string
}

func NewDockerManager(opts *config.Config) (manager *DockerManager, err error) {
	manager = new(DockerManager)
	manager.DockerManagerBase.init(opts)
	manager.registryName = fmt.Sprintf("%s:%s", opts.Registry_host, opts.Registry_port)
	return manager, nil
}

func (dm *DockerManager) Create(name string, sandbox_dir string) (sb.Sandbox, error) {
	volumes := []string{
		fmt.Sprintf("%s:%s", sandbox_dir, "/host/")}

	sandbox, err := dm.create(name, sandbox_dir, name, volumes)
	if err != nil {
		return nil, err
	}

	return sandbox, nil
}

func (dm *DockerManager) Pull(name string) error {
	// delete if it exists, so we can pull a new one
	imgExists, err := dm.DockerImageExists(name)
	if err != nil {
		return err
	}
	if imgExists {
		if dm.opts.Skip_pull_existing {
			return nil
		}
		opts := docker.RemoveImageOptions{Force: true}
		if err := dm.client().RemoveImageExtended(name, opts); err != nil {
			return err
		}
	}

	// pull new code
	if err := dm.dockerPull(name); err != nil {
		return err
	}

	return nil
}

func (dm *DockerManager) dockerPull(img string) error {
	err := dm.client().PullImage(
		docker.PullImageOptions{
			Repository: dm.registryName + "/" + img,
			Registry:   dm.registryName,
			Tag:        "latest",
		},
		docker.AuthConfiguration{},
	)

	if err != nil {
		return fmt.Errorf("failed to pull '%v' from %v registry\n", img, dm.registryName)
	}

	err = dm.client().TagImage(
		dm.registryName+"/"+img,
		docker.TagImageOptions{Repo: img, Force: true})
	if err != nil {
		log.Printf("failed to re-tag container: %v\n", err)
		return fmt.Errorf("failed to re-tag container: %v\n", err)
	}

	return nil
}

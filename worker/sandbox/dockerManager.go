package sandbox

import (
	"fmt"
	"log"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/phonyphonecall/turnip"
)

type DockerManager struct {
	// private
	registryName string
	opts         *config.Config

	// public
	dClient  *docker.Client
	createT  *turnip.Turnip
	pauseT   *turnip.Turnip
	unpauseT *turnip.Turnip
	pullT    *turnip.Turnip
	restartT *turnip.Turnip
	inspectT *turnip.Turnip
	startT   *turnip.Turnip
	removeT  *turnip.Turnip
	logT     *turnip.Turnip
}

func NewDockerManager(opts *config.Config) (manager *DockerManager) {
	manager = new(DockerManager)

	// NOTE: This requires that users have pre-configured the environement a docker daemon
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		manager.dClient = c
	}

	manager.opts = opts
	manager.registryName = fmt.Sprintf("%s:%s", opts.Registry_host, opts.Registry_port)
	manager.initTimers()
	return manager
}

// use "UploadToContainer"
func (dm *DockerManager) Create(name string) (Sandbox, error) {
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "0"}}}
	labels := map[string]string{"openlambda.cluster": dm.opts.Cluster_name}

	log.Printf("Use CLUSTER = '%v'\n", dm.opts.Cluster_name)

	container, err := dm.dClient.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:        name,
				AttachStdout: true,
				AttachStderr: true,
				ExposedPorts: internalAppPort,
				Labels:       labels,
			},
			HostConfig: &docker.HostConfig{
				PortBindings:    portBindings,
				PublishAllPorts: true,
			},
		},
	)

	if err != nil {
		return nil, err
	}

	sandbox := &DockerSandbox{name: name, container: container, mgr: dm}
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
		if err := dm.dClient.RemoveImageExtended(name, opts); err != nil {
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
	dm.pullT.Start()
	err := dm.dClient.PullImage(
		docker.PullImageOptions{
			Repository: dm.registryName + "/" + img,
			Registry:   dm.registryName,
			Tag:        "latest",
		},
		docker.AuthConfiguration{},
	)
	dm.pullT.Stop()

	if err != nil {
		return fmt.Errorf("failed to pull '%v' from %v registry\n", img, dm.registryName)
	}

	err = dm.dClient.TagImage(
		dm.registryName+"/"+img,
		docker.TagImageOptions{Repo: img, Force: true})
	if err != nil {
		log.Printf("failed to re-tag container: %v\n", err)
		return fmt.Errorf("failed to re-tag container: %v\n", err)
	}

	return nil
}

// Left public for handler tests. Consider refactor
func (dm *DockerManager) DockerImageExists(img_name string) (bool, error) {
	_, err := dm.dClient.InspectImage(img_name)
	if err == docker.ErrNoSuchImage {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (dm *DockerManager) Dump() {
	opts := docker.ListContainersOptions{All: true}
	containers, err := dm.dClient.ListContainers(opts)
	if err != nil {
		log.Fatal("Could not get container list")
	}
	log.Printf("=====================================\n")
	for idx, info := range containers {
		container, err := dm.dClient.InspectContainer(info.ID)
		if err != nil {
			log.Fatal("Could not get container")
		}

		log.Printf("CONTAINER %d: %v, %v, %v\n", idx,
			info.Image,
			container.ID[:8],
			container.State.String())
	}
	log.Printf("=====================================\n")
	log.Println()
	log.Printf("====== Docker Operation Stats =======\n")
	log.Printf("\tcreate: \t%fms\n", dm.createT.AverageMs())
	log.Printf("\tinspect: \t%fms\n", dm.inspectT.AverageMs())
	log.Printf("\tlogs: \t%fms\n", dm.logT.AverageMs())
	log.Printf("\tpause: \t\t%fms\n", dm.pauseT.AverageMs())
	log.Printf("\tpull: \t\t%fms\n", dm.pullT.AverageMs())
	log.Printf("\tremove: \t%fms\n", dm.removeT.AverageMs())
	log.Printf("\trestart: \t%fms\n", dm.restartT.AverageMs())
	log.Printf("\trestart: \t%fms\n", dm.restartT.AverageMs())
	log.Printf("\tunpause: \t%fms\n", dm.unpauseT.AverageMs())
	log.Printf("=====================================\n")
}

func (dm *DockerManager) initTimers() {
	dm.createT = turnip.NewTurnip()
	dm.inspectT = turnip.NewTurnip()
	dm.pauseT = turnip.NewTurnip()
	dm.pullT = turnip.NewTurnip()
	dm.removeT = turnip.NewTurnip()
	dm.restartT = turnip.NewTurnip()
	dm.startT = turnip.NewTurnip()
	dm.unpauseT = turnip.NewTurnip()
	dm.logT = turnip.NewTurnip()
}

func (dm *DockerManager) client() *docker.Client {
	return dm.dClient
}

func (dm *DockerManager) createTimer() *turnip.Turnip {
	return dm.createT
}

func (dm *DockerManager) inspectTimer() *turnip.Turnip {
	return dm.inspectT
}

func (dm *DockerManager) pauseTimer() *turnip.Turnip {
	return dm.pauseT
}

func (dm *DockerManager) pullTimer() *turnip.Turnip {
	return dm.pullT
}

func (dm *DockerManager) removeTimer() *turnip.Turnip {
	return dm.removeT
}

func (dm *DockerManager) restartTimer() *turnip.Turnip {
	return dm.restartT
}

func (dm *DockerManager) startTimer() *turnip.Turnip {
	return dm.startT
}

func (dm *DockerManager) unpauseTimer() *turnip.Turnip {
	return dm.unpauseT
}

func (dm *DockerManager) logTimer() *turnip.Turnip {
	return dm.logT
}

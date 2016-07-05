package sandbox

import (
	"fmt"
	"log"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/phonyphonecall/turnip"
)

type DockerManager struct {
	client *docker.Client

	registryName string
	opts         *config.Config

	// timers
	createTimer  *turnip.Turnip
	pauseTimer   *turnip.Turnip
	unpauseTimer *turnip.Turnip
	pullTimer    *turnip.Turnip
	restartTimer *turnip.Turnip
	inspectTimer *turnip.Turnip
	startTimer   *turnip.Turnip
	removeTimer  *turnip.Turnip
	logTimer     *turnip.Turnip
}

func NewDockerManager(opts *config.Config) (manager *DockerManager) {
	manager = new(DockerManager)

	// NOTE: This requires that users have pre-configured the environement a docker daemon
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		manager.client = c
	}

	manager.opts = opts
	manager.registryName = fmt.Sprintf("%s:%s", opts.Registry_host, opts.Registry_port)
	manager.initTimers()
	return manager
}

func (dm *DockerManager) Create(name string) (Sandbox, error) {
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "0"}}}

	container, err := dm.client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:        name,
				AttachStdout: true,
				AttachStderr: true,
				ExposedPorts: internalAppPort,
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
		if err := dm.client.RemoveImageExtended(name, opts); err != nil {
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
	dm.pullTimer.Start()
	err := dm.client.PullImage(
		docker.PullImageOptions{
			Repository: dm.registryName + "/" + img,
			Registry:   dm.registryName,
			Tag:        "latest",
		},
		docker.AuthConfiguration{},
	)
	dm.pullTimer.Stop()

	if err != nil {
		return fmt.Errorf("failed to pull '%v' from %v registry\n", img, dm.registryName)
	}

	err = dm.client.TagImage(
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
	_, err := dm.client.InspectImage(img_name)
	if err == docker.ErrNoSuchImage {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (dm *DockerManager) Dump() {
	opts := docker.ListContainersOptions{All: true}
	containers, err := dm.client.ListContainers(opts)
	if err != nil {
		log.Fatal("Could not get container list")
	}
	log.Printf("=====================================\n")
	for idx, info := range containers {
		container, err := dm.client.InspectContainer(info.ID)
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
	log.Printf("\tcreate: \t%fms\n", dm.createTimer.AverageMs())
	log.Printf("\tinspect: \t%fms\n", dm.inspectTimer.AverageMs())
	log.Printf("\tlogs: \t%fms\n", dm.logTimer.AverageMs())
	log.Printf("\tpause: \t\t%fms\n", dm.pauseTimer.AverageMs())
	log.Printf("\tpull: \t\t%fms\n", dm.pullTimer.AverageMs())
	log.Printf("\tremove: \t%fms\n", dm.removeTimer.AverageMs())
	log.Printf("\trestart: \t%fms\n", dm.restartTimer.AverageMs())
	log.Printf("\trestart: \t%fms\n", dm.restartTimer.AverageMs())
	log.Printf("\tunpause: \t%fms\n", dm.unpauseTimer.AverageMs())
	log.Printf("=====================================\n")
}

func (dm *DockerManager) Client() *docker.Client {
	return dm.client
}

func (dm *DockerManager) initTimers() {
	dm.createTimer = turnip.NewTurnip()
	dm.inspectTimer = turnip.NewTurnip()
	dm.pauseTimer = turnip.NewTurnip()
	dm.pullTimer = turnip.NewTurnip()
	dm.removeTimer = turnip.NewTurnip()
	dm.restartTimer = turnip.NewTurnip()
	dm.startTimer = turnip.NewTurnip()
	dm.unpauseTimer = turnip.NewTurnip()
	dm.logTimer = turnip.NewTurnip()
}

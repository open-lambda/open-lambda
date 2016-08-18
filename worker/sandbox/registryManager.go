package sandbox

import (
	"bytes"
	"log"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/phonyphonecall/turnip"

	docker "github.com/fsouza/go-dockerclient"
	r "github.com/open-lambda/open-lambda/worker/registry"
)

type RegistryManager struct {
	// private
	opts     *config.Config
	reg      *r.PullClient
	handlers map[string][]byte

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

func NewRegistryManager(opts *config.Config) (manager *RegistryManager) {
	manager = new(RegistryManager)

	// NOTE: This requires that users have pre-configured the environement a docker daemon
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		manager.dClient = c
	}

	// TODO: trace config back to properly input cluster addresses
	manager.reg = r.InitPullClient(opts.Reg_cluster)
	manager.opts = opts
	manager.initTimers()
	return manager
}

func (dm *RegistryManager) Create(name string) (Sandbox, error) {
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "0"}}}
	labels := map[string]string{"openlambda.cluster": dm.opts.Cluster_name}

	log.Printf("Use CLUSTER = '%v'\n", dm.opts.Cluster_name)

	container, err := dm.dClient.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:        "ubuntu:trusty", //TODO
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

	r := bytes.NewReader(dm.handlers[name])

	err = dm.dClient.UploadToContainer(container.ID,
		docker.UploadToContainerOptions{
			InputStream:          r,
			Path:                 "/",
			NoOverwriteDirNonDir: false, // TODO
		},
	)

	if err != nil {
		return nil, err
	}

	sandbox := &DockerSandbox{name: name, container: container, mgr: dm}
	return sandbox, nil
}

// TODO: evict code (onto disk?)
func (dm *RegistryManager) Pull(name string) error {
	dm.handlers[name] = dm.reg.Pull(name)
	// TODO: fix error handling
	return nil
}

func (dm *RegistryManager) Dump() {
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

func (dm *RegistryManager) initTimers() {
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

func (dm *RegistryManager) client() *docker.Client {
	return dm.dClient
}

func (dm *RegistryManager) createTimer() *turnip.Turnip {
	return dm.createT
}

func (dm *RegistryManager) inspectTimer() *turnip.Turnip {
	return dm.inspectT
}

func (dm *RegistryManager) pauseTimer() *turnip.Turnip {
	return dm.pauseT
}

func (dm *RegistryManager) pullTimer() *turnip.Turnip {
	return dm.pullT
}

func (dm *RegistryManager) removeTimer() *turnip.Turnip {
	return dm.removeT
}

func (dm *RegistryManager) restartTimer() *turnip.Turnip {
	return dm.restartT
}

func (dm *RegistryManager) startTimer() *turnip.Turnip {
	return dm.startT
}

func (dm *RegistryManager) unpauseTimer() *turnip.Turnip {
	return dm.unpauseT
}

func (dm *RegistryManager) logTimer() *turnip.Turnip {
	return dm.logT
}

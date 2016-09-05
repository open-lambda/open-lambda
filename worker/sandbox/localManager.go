package sandbox

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/phonyphonecall/turnip"

	docker "github.com/fsouza/go-dockerclient"
)

type LocalManager struct {
	// private
	opts        *config.Config
	handler_dir string

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

func NewLocalManager(opts *config.Config) (manager *LocalManager) {
	manager = new(LocalManager)

	// NOTE: This requires that users have pre-configured the environement a docker daemon
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		manager.dClient = c
	}

	manager.opts = opts
	manager.initTimers()
	manager.handler_dir = opts.Reg_dir

	return manager
}

func (lm *LocalManager) Create(name string) (Sandbox, error) {
	internalAppPort := map[docker.Port]struct{}{"8080/tcp": {}}
	portBindings := map[docker.Port][]docker.PortBinding{
		"8080/tcp": {{HostIP: "0.0.0.0", HostPort: "0"}}}
	labels := map[string]string{"openlambda.cluster": lm.opts.Cluster_name}

	log.Printf("Use CLUSTER = '%v'\n", lm.opts.Cluster_name)

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
	log.Printf("=====================================\n")
	log.Println()
	log.Printf("====== Docker Operation Stats =======\n")
	log.Printf("\tcreate: \t%fms\n", lm.createT.AverageMs())
	log.Printf("\tinspect: \t%fms\n", lm.inspectT.AverageMs())
	log.Printf("\tlogs: \t%fms\n", lm.logT.AverageMs())
	log.Printf("\tpause: \t\t%fms\n", lm.pauseT.AverageMs())
	log.Printf("\tpull: \t\t%fms\n", lm.pullT.AverageMs())
	log.Printf("\tremove: \t%fms\n", lm.removeT.AverageMs())
	log.Printf("\trestart: \t%fms\n", lm.restartT.AverageMs())
	log.Printf("\trestart: \t%fms\n", lm.restartT.AverageMs())
	log.Printf("\tunpause: \t%fms\n", lm.unpauseT.AverageMs())
	log.Printf("=====================================\n")
}

func (lm *LocalManager) initTimers() {
	lm.createT = turnip.NewTurnip()
	lm.inspectT = turnip.NewTurnip()
	lm.pauseT = turnip.NewTurnip()
	lm.pullT = turnip.NewTurnip()
	lm.removeT = turnip.NewTurnip()
	lm.restartT = turnip.NewTurnip()
	lm.startT = turnip.NewTurnip()
	lm.unpauseT = turnip.NewTurnip()
	lm.logT = turnip.NewTurnip()
}

func (lm *LocalManager) client() *docker.Client {
	return lm.dClient
}

func (lm *LocalManager) createTimer() *turnip.Turnip {
	return lm.createT
}

func (lm *LocalManager) inspectTimer() *turnip.Turnip {
	return lm.inspectT
}

func (lm *LocalManager) pauseTimer() *turnip.Turnip {
	return lm.pauseT
}

func (lm *LocalManager) pullTimer() *turnip.Turnip {
	return lm.pullT
}

func (lm *LocalManager) removeTimer() *turnip.Turnip {
	return lm.removeT
}

func (lm *LocalManager) restartTimer() *turnip.Turnip {
	return lm.restartT
}

func (lm *LocalManager) startTimer() *turnip.Turnip {
	return lm.startT
}

func (lm *LocalManager) unpauseTimer() *turnip.Turnip {
	return lm.unpauseT
}

func (lm *LocalManager) logTimer() *turnip.Turnip {
	return lm.logT
}

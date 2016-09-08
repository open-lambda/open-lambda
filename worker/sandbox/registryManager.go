package sandbox

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/phonyphonecall/turnip"

	docker "github.com/fsouza/go-dockerclient"
	r "github.com/open-lambda/open-lambda/registry/src"
)

type RegistryManager struct {
	// private
	opts        *config.Config
	reg         *r.PullClient
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

func NewRegistryManager(opts *config.Config) (manager *RegistryManager) {
	manager = new(RegistryManager)

	// NOTE: This requires that users have pre-configured the environement a docker daemon
	if c, err := docker.NewClientFromEnv(); err != nil {
		log.Fatal("failed to get docker client: ", err)
	} else {
		manager.dClient = c
	}

	manager.reg = r.InitPullClient(opts.Reg_cluster)
	manager.opts = opts
	manager.initTimers()
	manager.handler_dir = "/tmp/olhandlers/"
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
	labels := map[string]string{"openlambda.cluster": rm.opts.Cluster_name}

	handler := filepath.Join(rm.handler_dir, name)
	volumes := []string{fmt.Sprintf("%s:%s", handler, "/handler/")}

	container, err := rm.dClient.CreateContainer(
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

func (rm *RegistryManager) Dump() {
	opts := docker.ListContainersOptions{All: true}
	containers, err := rm.dClient.ListContainers(opts)
	if err != nil {
		log.Fatal("Could not get container list")
	}
	log.Printf("=====================================\n")
	for idx, info := range containers {
		container, err := rm.dClient.InspectContainer(info.ID)
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
	log.Printf("\tcreate: \t%fms\n", rm.createT.AverageMs())
	log.Printf("\tinspect: \t%fms\n", rm.inspectT.AverageMs())
	log.Printf("\tlogs: \t%fms\n", rm.logT.AverageMs())
	log.Printf("\tpause: \t\t%fms\n", rm.pauseT.AverageMs())
	log.Printf("\tpull: \t\t%fms\n", rm.pullT.AverageMs())
	log.Printf("\tremove: \t%fms\n", rm.removeT.AverageMs())
	log.Printf("\trestart: \t%fms\n", rm.restartT.AverageMs())
	log.Printf("\trestart: \t%fms\n", rm.restartT.AverageMs())
	log.Printf("\tunpause: \t%fms\n", rm.unpauseT.AverageMs())
	log.Printf("=====================================\n")
}

func (rm *RegistryManager) initTimers() {
	rm.createT = turnip.NewTurnip()
	rm.inspectT = turnip.NewTurnip()
	rm.pauseT = turnip.NewTurnip()
	rm.pullT = turnip.NewTurnip()
	rm.removeT = turnip.NewTurnip()
	rm.restartT = turnip.NewTurnip()
	rm.startT = turnip.NewTurnip()
	rm.unpauseT = turnip.NewTurnip()
	rm.logT = turnip.NewTurnip()
}

func (rm *RegistryManager) client() *docker.Client {
	return rm.dClient
}

func (rm *RegistryManager) createTimer() *turnip.Turnip {
	return rm.createT
}

func (rm *RegistryManager) inspectTimer() *turnip.Turnip {
	return rm.inspectT
}

func (rm *RegistryManager) pauseTimer() *turnip.Turnip {
	return rm.pauseT
}

func (rm *RegistryManager) pullTimer() *turnip.Turnip {
	return rm.pullT
}

func (rm *RegistryManager) removeTimer() *turnip.Turnip {
	return rm.removeT
}

func (rm *RegistryManager) restartTimer() *turnip.Turnip {
	return rm.restartT
}

func (rm *RegistryManager) startTimer() *turnip.Turnip {
	return rm.startT
}

func (rm *RegistryManager) unpauseTimer() *turnip.Turnip {
	return rm.unpauseT
}

func (rm *RegistryManager) logTimer() *turnip.Turnip {
	return rm.logT
}

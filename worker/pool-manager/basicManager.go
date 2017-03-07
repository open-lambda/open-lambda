package pmanager

/*

Manages lambdas using the OpenLambda registry (built on RethinkDB).

Creates lambda containers using the generic base image defined in
dockerManagerBase.go (BASE_IMAGE).

Handler code is mapped into the container by attaching a directory
(<handler_dir>/<lambda_name>) when the container is started.

*/

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/config"
	dutil "github.com/open-lambda/open-lambda/worker/dockerutil"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type ForkServer struct {
	sockPath string
	packages []string
}

type BasicManager struct {
	servers []*ForkServer
	poolDir string
	cid     string
}

func NewBasicManager(opts *config.Config) (bm *BasicManager, err error) {
	poolDir := opts.Pool_dir
	numServers := opts.Num_forkservers

	cid, err := initPoolContainer(poolDir, opts.Cluster_name, numServers)
	if err != nil {
		return nil, err
	}

	servers := make([]*ForkServer, numServers, numServers)
	for k := 0; k < numServers; k++ {
		sockPath := fmt.Sprintf("%s/fs%d/fs.sock", poolDir, k)

		start := time.Now()
		// wait up to 5s for server to initialize
		for os.IsNotExist(err) {
			_, err = os.Stat(sockPath)
			if time.Since(start).Seconds() > 5 {
				return nil, errors.New("forkservers failed to initialize")
			}
		}

		servers[k] = &ForkServer{
			sockPath: sockPath,
			packages: []string{},
		}
	}

	bm = &BasicManager{
		servers: servers,
		cid:     cid,
	}

	return bm, nil
}

func (bm *BasicManager) ForkEnter(sandbox sb.Sandbox) (err error) {
	fs := bm.chooseRandom()

	docker_sb, ok := sandbox.(*sb.DockerSandbox)
	if !ok {
		return errors.New("forkenter only supported with DockerSandbox")
	}

	pid, err := sendFds(fs.sockPath, docker_sb.NSPid())
	if err != nil {
		return err
	}

	// change cgroup of spawned lambda server
	err = docker_sb.CGroupEnter(pid)
	if err != nil {
		return err
	}

	return nil
}

func initPoolContainer(poolDir, clusterName string, numServers int) (cid string, err error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return "", err
	}

	if err = os.MkdirAll(poolDir, os.ModeDir); err != nil {
		return "", err
	}

	labels := map[string]string{
		dutil.DOCKER_LABEL_CLUSTER: clusterName,
		dutil.DOCKER_LABEL_TYPE:    dutil.POOL,
	}

	volumes := []string{
		fmt.Sprintf("%s:%s", poolDir, "/host"),
	}

	caps := []string{"SYS_ADMIN"}

	cmd := []string{"python", "/initservers.py", fmt.Sprintf("%d", numServers)}

	container, err := client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:  dutil.POOL_IMAGE,
				Labels: labels,
				Cmd:    cmd,
			},
			HostConfig: &docker.HostConfig{
				Binds:   volumes,
				PidMode: "host",
				CapAdd:  caps,
			},
		},
	)

	if err := client.StartContainer(container.ID, nil); err != nil {
		return "", err
	}

	return container.ID, nil
}

func (bm *BasicManager) chooseRandom() (server *ForkServer) {
	rand.Seed(time.Now().Unix())
	k := rand.Int() % len(bm.servers)

	return bm.servers[k]
}

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
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/open-lambda/open-lambda/worker/config"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type ForkServer struct {
	//packages []string TODO
	sockPath string
}

type BasicManager struct {
	servers []*ForkServer
}

func NewForkServer(sockPath string) (fs *ForkServer, err error) {
	if err = runLambdaServer(sockPath); err != nil {
		return nil, err
	}

	fs = &ForkServer{sockPath: sockPath}

	return fs, nil
}

func NewBasicManager(opts *config.Config) (bm *BasicManager, err error) {
	sockDir := "/var/tmp/olsocks"
	if err = os.MkdirAll(sockDir, os.ModeDir); err != nil {
		return nil, err
	}

	numServers := opts.Num_forkservers
	servers := make([]*ForkServer, numServers, numServers)
	for k := 0; k < numServers; k++ {
		sockPath := filepath.Join(sockDir, fmt.Sprintf("ol-%d.sock", k))
		if err != nil {
			return nil, err
		}

		fs, err := NewForkServer(sockPath)
		if err != nil {
			return nil, err
		}

		servers[k] = fs
	}

	// TODO: find better way to wait for lambda server initialization
	time.Sleep(time.Second)

	bm = &BasicManager{
		servers: servers,
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

func (bm *BasicManager) chooseRandom() (server *ForkServer) {
	rand.Seed(time.Now().Unix())
	k := rand.Int() % len(bm.servers)

	return bm.servers[k]
}

/* Start the lambda python server, listening on socket at sockPath */
func runLambdaServer(sockPath string) (err error) {
	_, absPath, _, _ := runtime.Caller(1)
	relPath := "../../../../../../../../../lambda/server.py" // disgusting path from this file in hack dir to server script
	serverPath := filepath.Join(absPath, relPath)

	cmd := exec.Command("/usr/bin/python", serverPath, sockPath)
	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}

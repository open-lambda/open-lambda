package pmanager

/*

Manages lambdas using the OpenLambda registry (built on RethinkDB).

Creates lambda containers using the generic base image defined in
dockerManagerBase.go (BASE_IMAGE).

Handler code is mapped into the container by attaching a directory
(<handler_dir>/<lambda_name>) when the container is started.

*/

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/tv42/httpunix"

	"github.com/open-lambda/open-lambda/worker/config"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type ForkServer struct {
	packages []string
	client   *http.Client
}

type BasicManager struct {
	servers []ForkServer
}

func NewForkServer(sock_path string) (fs *ForkServer, err error) {
	err = runLambdaServer(sock_path)
	if err != nil {
		return nil, err
	}

	t := &httpunix.Transport{
		DialTimeout:           100 * time.Millisecond,
		RequestTimeout:        1 * time.Second,
		ResponseHeaderTimeout: 1 * time.Second,
	}
	// registers a URL location? - first param probably wrong
	t.RegisterLocation("forkenter", sock_path)

	fs = new(ForkServer)
	fs.client = &http.Client{
		Transport: u,
	}

	return fs, nil
}

func NewBasicManager(opts *config.Config) (bm *BasicManager, err error) {
	sock_dir := "/olsocks"
	err = os.Mkdir(sock_dir, os.ModeDir)
	if err != nil {
		return nil, err
	}

	// TODO: make number of servers configurable
	var servers [3]ForkServer
	for k := range 5 {
		sock_path := filepath.join(sock_dir, fmt.Sprintf("ol-%d.sock", k))
		sock_file, err := os.OpenFile(sock_path, os.O_RDWR|os.O_CREATE, os.ModeSocket)
		if err != nil {
			return nil, err
		}

		servers[k] = NewForkServer(sock_path)

	}

	bm = new(BasicManager)
	bm.servers = servers

	return bm, nil
}

func (bm *BasicManager) ForkEnter(sandbox, sandbox_dir) (err error) {
	sock_path := filepath.join(sandbox_dir, "ol.sock")
	_, err := os.OpenFile(sock_path, os.O_RDWR|os.O_CREATE, os.ModeSocket)
	if err != nil {
		return err
	}

	fs := bm.chooseRandom()

	body := map[string]string{"nspid": sandbox.nspid, "sock_file": sock_path}
	json_body, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := client.Post("http+unix://forkenter", "application/json", bytes.NewBuffer(json_body))
	if err != nil {
		return err
	}

	//TODO: check response?
}

func (bm *BasicManager) chooseRandom() (server *ForkServer) {
	rand.Seed(time.Now().Unix())
	n := rand.Int() % len(bm.servers)

	return servers[n]
}

func runLambdaServer(sock_path string) (err error) {
	//TODO
}

package pmanager

/*

Manages lambdas using the OpenLambda registry (built on RethinkDB).

Creates lambda containers using the generic base image defined in
dockerManagerBase.go (BASE_IMAGE).

Handler code is mapped into the container by attaching a directory
(<handler_dir>/<lambda_name>) when the container is started.

*/

import (
	"fmt"
    "time"
    "runtime"
	"os"
	"os/exec"
	"math/rand"
	"path/filepath"

	"github.com/open-lambda/open-lambda/worker/config"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type ForkServer struct {
	//packages []string TODO
	fifo     *os.File
}

type BasicManager struct {
	servers []*ForkServer
}

func NewForkServer(fifoPath string) (fs *ForkServer, err error) {
    fifo, err := os.OpenFile(fifoPath, os.O_CREATE|os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return nil, err
	}

    if err = runLambdaServer(fifoPath); err != nil {
        return nil, err
    }

    fs = &ForkServer{fifo: fifo}

	return fs, nil
}

func NewBasicManager(opts *config.Config) (bm *BasicManager, err error) {
	fifoDir := "/tmp/olpipes" // TODO: tmp?
    if _, err = os.Stat(fifoDir); os.IsNotExist(err) {
        if err = os.Mkdir(fifoDir, os.ModeDir); err != nil {
            return nil, err
        }
    }

	// TODO: make number of servers configurable
	servers := make([]*ForkServer, 5, 5)
	for k := 0; k < 5; k++ {
		fifoPath := filepath.Join(fifoDir, fmt.Sprintf("ol-%d.pipe", k))
		if err != nil {
			return nil, err
		}

        fs, err := NewForkServer(fifoPath)
        if err != nil {
            return nil, err
        }

		servers[k] = fs
	}

    bm = &BasicManager{
        servers: servers,
    }

	return bm, nil
}

func (bm *BasicManager) ForkEnter(sandbox sb.Sandbox) (err error) {
	fs := bm.chooseRandom()

    msg := fmt.Sprintf("%d", sandbox.NSPid())
    if _, err := fs.fifo.WriteString(msg); err != nil{
        return err
    }

	//TODO: respond?
    return nil
}

func (bm *BasicManager) chooseRandom() (server *ForkServer) {
    rand.Seed(time.Now().Unix())
    k := rand.Int() % len(bm.servers)

	return bm.servers[k]
}

// start the python interpreter, listening on passed pipe
func runLambdaServer(fifoPath string) (err error) {
    _, absPath, _, _ := runtime.Caller(1)
    relPath := "../../../../../../../../../lambda/server.py" // disgusting path from this file in hack to server script
    serverPath := filepath.Join(absPath, relPath)

    cmd := exec.Command("/usr/bin/python", serverPath, fifoPath)
    if err := cmd.Start(); err != nil {
        return err
    }

    return nil
}

package cloudvm

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"os/exec"
)

type LocalWorkerPool struct {
	// no platform specific attributes
}

type LocalWorker struct {
	Worker
	workerProcess *exec.Cmd
	stdinPipe     *os.File      // Pipe for sending tasks to the process
	stdoutPipe    *bufio.Reader // Pipe for reading output from the process
}

func NewLocalWorkerPool() *WorkerPool {
	return &WorkerPool{
		WorkerPoolPlatform: &LocalWorkerPool{},
	}
}

func (_ *LocalWorkerPool) NewWorker(workerId string) *LocalWorker {
	return &LocalWorker{
		Worker: Worker{
			workerId: workerId,
		},
		workerProcess: nil, // No process started yet
	}
}

func (_ *LocalWorkerPool) CreateInstance(worker *LocalWorker) {
	log.Printf("created new local worker: %s\n", worker.workerId)
}

func (_ *LocalWorkerPool) DeleteInstance(worker *LocalWorker) {
	log.Printf("deleted local worker: %s\n", worker.workerId)
}

func (_ *LocalWorkerPool) ForwardTask(w http.ResponseWriter, _ *http.Request, worker *LocalWorker) {
	log.Printf("forwarding task")
}

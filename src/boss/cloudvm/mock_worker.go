package cloudvm

import (
	"fmt"
	"log"
	"net/http"
)

// WORKER IMPLEMENTATION: MockWorker
type MockWorkerPool struct {
	//no platform specific attributes
}

type MockWorker struct {
	//no platform specific attributes
}

func NewMockWorkerPool() *WorkerPool {
	return &WorkerPool{
		WorkerPoolPlatform: &MockWorkerPool{},
	}
}

func (_ *MockWorkerPool) NewWorker(workerId string) *Worker {
	return &Worker{
		workerId: workerId,
		workerIp: "",
	}
}

func (_ *MockWorkerPool) CreateInstance(worker *Worker) {
	log.Printf("created new mock worker: %s\n", worker.workerId)
}

func (_ *MockWorkerPool) DeleteInstance(worker *Worker) {
	log.Printf("deleted mock worker: %s\n", worker.workerId)
}

func (_ *MockWorkerPool) ForwardTask(w http.ResponseWriter, _ *http.Request, worker *Worker) {
	s := fmt.Sprintf("hello from %s\n", worker.workerId)
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(s))
	if err != nil {
		panic(err)
	}
}

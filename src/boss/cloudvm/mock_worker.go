package cloudvm

import (
	"fmt"
	"log"
	"net/http"
)

// WORKER IMPLEMENTATION: MockWorker
type MockWorkerPoolPlatform struct {
	// no platform specific attributes
}

type MockWorker struct {
	// no platform specific attributes
}

func NewMockWorkerPool() *WorkerPool {
	return &WorkerPool{
		WorkerPoolPlatform: &MockWorkerPoolPlatform{},
	}
}

func (_ *MockWorkerPoolPlatform) NewWorker(workerId string) *Worker {
	return &Worker{
		workerId: workerId,
		host:     "",
	}
}

func (_ *MockWorkerPoolPlatform) CreateInstance(worker *Worker) {
	log.Printf("created new mock worker: %s\n", worker.workerId)
}

func (_ *MockWorkerPoolPlatform) DeleteInstance(worker *Worker) {
	log.Printf("deleted mock worker: %s\n", worker.workerId)
}

func (_ *MockWorkerPoolPlatform) ForwardTask(w http.ResponseWriter, _ *http.Request, worker *Worker) {
	s := fmt.Sprintf("hello from %s\n", worker.workerId)
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(s))
	if err != nil {
		panic(err)
	}
}

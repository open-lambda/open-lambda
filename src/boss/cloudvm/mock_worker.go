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

func (pool *MockWorkerPool) NewWorker(workerId string) *Worker {
	return &Worker{
		workerId:	workerId,
		workerIp:	"",
	}
}

func (pool *MockWorkerPool) CreateInstance(worker *Worker) {
	log.Printf("created new mock worker: %s\n", worker.workerId)
}

func (pool *MockWorkerPool) DeleteInstance(worker *Worker) {
	log.Printf("deleted mock worker: %s\n", worker.workerId)
}

func (pool *MockWorkerPool) ForwardTask(w http.ResponseWriter, r *http.Request, worker *Worker) {
	s := fmt.Sprintf("hello from %s\n", worker.workerId)
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(s))
	if err != nil {
		panic(err)
	}
}
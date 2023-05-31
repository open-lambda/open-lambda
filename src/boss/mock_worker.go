package boss

import (
	"log"
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
		workerId: workerId,
		workerIp: "",
	}
}

func (pool *MockWorkerPool) CreateInstance(worker *Worker) {
	log.Printf("created new mock worker: %s\n", worker.workerId)
}

func (pool *MockWorkerPool) DeleteInstance(worker *Worker) {
	log.Printf("deleted mock worker: %s\n", worker.workerId)
}

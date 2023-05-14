package boss

import (
	"fmt"
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

func (pool *MockWorkerPool) NewWorker(nextId int) *Worker {
	workerId := fmt.Sprintf("worker-%d", nextId)
	return &Worker{
		workerId:       workerId,
		workerIp:       "",
		WorkerPlatform: MockWorker{},
	}
}

func (pool *MockWorkerPool) CreateInstance(worker *Worker) {
	log.Printf("created new mock worker: %s\n", worker.workerId)
}

func (pool *MockWorkerPool) DeleteInstance(worker *Worker) {
	log.Printf("deleted mock worker: %s\n", worker.workerId)
}

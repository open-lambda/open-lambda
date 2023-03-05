package boss

import (
	"fmt"
	"log"
)

// WORKER IMPLEMENTATION: MockWorker
type MockWorkerPool struct {
	//no additional attributes yet
}

type MockWorker struct {
	//no additional attributes yet
}

func NewMockWorkerPool() (*WorkerPool, error) {
	return &WorkerPool {
		nextId:	1,
		workers: map[string]*Worker{},
		queue: make(chan *Worker, Conf.Worker_Cap),
		WorkerPoolPlatform: &MockWorkerPool {},
	}, nil
}

func (pool *MockWorkerPool) NewWorker(nextId int) *Worker {
	workerId := fmt.Sprintf("worker-%d", nextId)
   return &Worker{
	   workerId: workerId,
	   workerIp: "",
	   isIdle: true,
	   WorkerPlatform: MockWorker{},
   }
}

func (pool *MockWorkerPool) CreateInstance(worker *Worker) {
	log.Println("created new mock worker: %s\n", worker.workerId)
}

func (pool *MockWorkerPool) DeleteInstance(worker *Worker) {
	log.Println("delete mock worker: %s\n", worker.workerId)
}
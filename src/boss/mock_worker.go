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

func NewMockWorkerPool() (*WorkerPool, error) {
	return &WorkerPool {
		nextId:	1, //this should be similar among all platform
		workers: map[string]*Worker{}, //this should be similar among all platform
		queue: make(chan *Worker, Conf.Worker_Cap), //this should be similar among all platform
		WorkerPoolPlatform: &MockWorkerPool {},
	}, nil

	//WorkerPoolPlatform should be platform specific workerpool struct (MockWorkerPool, GcpWorkerPool, AzureWorkerPool)
}

func (pool *MockWorkerPool) NewWorker(nextId int) *Worker {
	workerId := fmt.Sprintf("worker-%d", nextId)
   return &Worker{
	   workerId: workerId, //this should be similar among all platform
	   workerIp: "", //initialize this to empty string and modify ip after new vm instance has been created
	   isIdle: true, //this should be similar among all platform
	   WorkerPlatform: MockWorker{},
   }

   //Equivalent to CreateWorker() function in previous design with slightly different Worker struct
   //WorkerPlatform should be platform specific worker struct (MockWorker, GcpWorker, AzureWorker)
   //But, do not call go worker.launch() and go worker.task()
   //you don't have to add worker to pool.workers or pool.queue
}

func (pool *MockWorkerPool) CreateInstance(worker *Worker) {
	//Equivalent to launch() function in previous design
	//set worker.workerIp after you created new instance

	log.Printf("created new mock worker: %s\n", worker.workerId)
}

func (pool *MockWorkerPool) DeleteInstance(worker *Worker) {
	//Equivalent to Close() function in previous design
	//1. ssh into the instance of given worker and kill worker
	//2. delete the instance
	//you don't have to remove worker from pool.workers or pool.queue
	log.Printf("deleted mock worker: %s\n", worker.workerId)
}
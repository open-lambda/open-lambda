package boss

import (
	"fmt"
	"log"
)

type AzureWorkerPool struct {
	nextId int
	//TODO: add additional field if needed for Azure
	workers map[string]Worker
}

type AzureWorker struct {
	pool *AzureWorkerPool
	workerId string
	workerIp string
	//TODO: add additional field if needed for Azure Worker
	reqChan  chan *Invocation
	exitChan chan bool
}

// WORKER IMPLEMENTATION: AzureWorker

func NewAzureWorkerPool() (*AzureWorkerPool, error) {
	//TODO: prepare for creating new vm:
	// - configure authentication, take snapshot of boss, etc

	pool := &AzureWorkerPool {
		nextId:		1,
		// add additional field if needed for Azure
		workers: map[string]Worker{},
	}
	return pool, nil
}

func (pool *AzureWorkerPool) CreateWorker(reqChan chan *Invocation) {
	log.Printf("creating azure worker")
	workerId := fmt.Sprintf("ol-worker-%d", pool.nextId) //TODO: give worker an id
	worker := &AzureWorker{
		pool: pool,
		workerId: workerId,
		//TODO: add additional field if needed for Azure Worker
		reqChan:  reqChan,
		exitChan: make(chan bool),
	}
	pool.nextId += 1

	go worker.launch() //Create new VM instance and Start Worker
	
	
	go worker.task() //go routine that forwards requests to worker VMs
	
	pool.workers[workerId] = worker
}

//delete worker with given workerId
//not used for current scaling down logic in boss
func (pool *AzureWorkerPool) DeleteWorker(workerId string) {
	pool.workers[workerId].Close()
}

//return list of active workers' ids
func (pool *AzureWorkerPool) Status() []string {
	var w = []string{}
	for k, _ := range pool.workers {
		w = append(w, k)
	}
	return w
}

//return number of active workers
func (pool *AzureWorkerPool) Size() int {
	return len(pool.workers)
}

//close all workers
//curl -X POST {boss ip}:5000/shutdown
func (pool *AzureWorkerPool) CloseAll() {
	for _, w := range pool.workers {
		w.Close() 
	}
}

func (worker *AzureWorker) launch() {
	//TODO: Create new VM instance and Start Worker
	//store vm instance's ip in worker.workerIp
}

func (worker *AzureWorker) task() {
	for {

		var req *Invocation
		select {
		case <-worker.exitChan: //not used for current scaling down logic
			return
		case req = <-worker.reqChan:
		}

		if req == nil { //when boss passes nil through request channel
						//any idle worker will receive this and closes itself
			worker.Close()
			return
		}

		err = forwardTask(req.w, req.r, worker.workerIp) //forward request to worker VM
		if err != nil {
			panic(err)
		}

		req.Done <- true
	}
}

func (worker *AzureWorker) Close() {
	select {
    case worker.exitChan <- true: //not used for current scaling down logic
    }

	log.Printf("stopping %s\n", worker.workerId)
	
	//TODO: stop or delete azure instance

	//delete(worker.pool.workers, worker.workerId)
}

package boss

import (
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
)

type WorkerPool struct {
	nextId  int
	workers map[string]*Worker //list of all workers
	queue   chan *Worker       //queue of all workers
	WorkerPoolPlatform //platform specific attributes and functions
}

type WorkerPoolPlatform interface {
	NewWorker(nextId int) *Worker  //return new worker struct
	CreateInstance(worker *Worker) //create new instance in the cloud platform
	DeleteInstance(worker *Worker) //delete cloud platform instance associated with give worker struct
}

type Worker struct {
	workerId       string
	workerIp       string
	numTask        int32 //count of outstanding tasks
	isKilled	   bool //true if to be killed
	WorkerPlatform //platform specific attributes and functions
}

type WorkerPlatform interface {
	//platform specific attributes and functions
	//do not require any functions yet
}

//return number of workers in the pool
func (pool *WorkerPool) Size() int {
	return len(pool.workers)
}

//add a new worker to the pool

// TODO: should we first create worker then push the worker to the idle queue?
// If the creation isn't finished yet and the boss send requests to that worker, it might cause problems.
// Maybe there's no need to have a "NewWorker" function? Just one "CreateInstance" might help.

// FIXEME: sometimes the azure part might fail due to cannot use the snapshot at the same time. But mostly it won't fail.

func (pool *WorkerPool) Scale(target int) error {
	for pool.Size() < target { // scale up
		nextId := pool.nextId
		pool.nextId += 1
		
		worker := pool.NewWorker(nextId)
	
		pool.workers[worker.workerId] = worker
		pool.queue <- worker
	
		go pool.CreateInstance(worker)
	}
	for pool.Size() > target { // scale down
		worker := <-pool.queue
		worker.isKilled = true
		delete(pool.workers, worker.workerId)
	}
	return nil
}

//run lambda function
func (pool *WorkerPool) RunLambda(w http.ResponseWriter, r *http.Request) {
	worker := <-pool.queue
	worker.numTask += 1
	pool.queue <- worker
	if Conf.Platform == "mock" {
		s := fmt.Sprintf("hello from %s\n", worker.workerId)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(s))
		if err != nil {
			panic(err)
		}
	} else {
		forwardTask(w, r, worker.workerIp)
	}

	if atomic.AddInt32(&worker.numTask, -1) == 0 && worker.isKilled {
		pool.DeleteInstance(worker)
	}

}

//return wokers' id and their status (idle/busy)
func (pool *WorkerPool) Status() []map[string]string {
	var w = []map[string]string{}

	for workerId, worker := range pool.workers {
		output := map[string]string{workerId: fmt.Sprintf("%d", worker.numTask)}
		w = append(w, output)
	}
	return w
}

//kill and delte all workers
func (pool *WorkerPool) Close() {
	for workerId, worker := range pool.workers {
		delete(pool.workers, workerId)
		pool.DeleteInstance(worker)
	}
}

// forward request to worker
func forwardTask(w http.ResponseWriter, req *http.Request, workerIp string) error {
	host := fmt.Sprintf("%s:%d", workerIp, 5000) //TODO: read from config
	req.URL.Scheme = "http"
	req.URL.Host = host
	req.Host = host
	req.RequestURI = ""

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return err
	}
	defer resp.Body.Close()

	io.Copy(w, resp.Body)

	return nil
}

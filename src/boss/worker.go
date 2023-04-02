package boss

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
)

const (
	Starting = iota
	Running
	Cleaning
	Destroying
)

type WorkerPool struct {
	nextId             int
	workers            map[string]*Worker //list of all workers
	queue              chan *Worker       //queue of all workers
	WorkerPoolPlatform                    //platform specific attributes and functions
	startingWorkers    map[string]*Worker
	runningWorkers     map[string]*Worker
	cleaningWorkers    map[string]*Worker
	destroyingWorkers  map[string]*Worker
	lock               sync.Mutex
	target             int
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
	isKilled       bool  //true if to be killed
	WorkerPlatform       //platform specific attributes and functions
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
	newNum := target - pool.Size()
	pool.target = target
	if newNum > 0 {
		scaleSize := newNum - len(pool.cleaningWorkers) - len(pool.destroyingWorkers)
		for i := 0; i < scaleSize; i++ { // scale up
			pool.lock.Lock()
			nextId := pool.nextId
			pool.nextId += 1
			worker := pool.NewWorker(nextId)
			pool.workers[worker.workerId] = worker
			pool.queue <- worker
			pool.startingWorkers[worker.workerId] = worker // add to starting map
			pool.lock.Unlock()

			go pool.CreateInstance(worker)
		}
	} else {
		delNum := 0 - newNum
		for i := 0; i < delNum; i++ { // scale down
			pool.lock.Lock()
			worker := <-pool.queue
			worker.isKilled = true
			pool.cleaningWorkers[worker.workerId] = worker // add to cleaning map
			pool.lock.Unlock()
			delete(pool.workers, worker.workerId)
		}
	}
	return nil
}

// a lock should be held here
// called when target has been changed
func (pool *WorkerPool) updateCluster(worker *Worker, evictedFrom int) bool {
	if evictedFrom == Cleaning {
		// check the target and running/starting
		if pool.Size() < pool.target {
			pool.startingWorkers[worker.workerId] = worker
			return true
		}
		return false
	}
	if evictedFrom == Destroying {
		if pool.Size() < pool.target {
			// start a new worker
			nextId := pool.nextId
			pool.nextId += 1
			worker := pool.NewWorker(nextId)
			pool.workers[worker.workerId] = worker
			pool.queue <- worker
			pool.startingWorkers[worker.workerId] = worker
			return true
		}
		return false
	}
	return false
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

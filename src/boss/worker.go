// BUG: cannot restart the worker, worker process xxxx does not a appear to be running, check worker.out
package boss

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Starting = iota
	Started
	Running
	Ran
	Cleaning
	Cleaned
	Destroying
	Destroyed
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
	startedWorker      *Worker
	ranWorker          *Worker
	cleanedWorker      *Worker
	needRestart        bool
	destroyedWorker    *Worker
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
	pool           *WorkerPool
}

type WorkerPlatform interface {
	//platform specific attributes and functions
	//do not require any functions yet
}

//return number of workers in the pool
func (pool *WorkerPool) Size() int {
	return len(pool.runningWorkers) + len(pool.startingWorkers)
}

func (pool *WorkerPool) Scale(target int) {
	pool.lock.Lock()
	pool.target = target
	pool.updateCluster()
	pool.lock.Unlock()
}

// lock is held before and after calling this function
// called when worker is been evicted from cleaning or destroying map
func (pool *WorkerPool) updateCluster() {
	newNum := pool.target - pool.Size()
	if newNum > 0 {
		scaleSize := newNum - len(pool.cleaningWorkers) - len(pool.destroyingWorkers)
		for i := 0; i < scaleSize; i++ { // scale up
			nextId := pool.nextId
			pool.nextId += 1
			worker := pool.NewWorker(nextId)
			pool.workers[worker.workerId] = worker
			pool.queue <- worker
			pool.startingWorkers[worker.workerId] = worker // add to starting map
			conf, err = ReadAzureConfig()
			if err != nil {
				log.Fatalf(err.Error())
			}

			go pool.CreateInstance(worker)
		}
	} else {
		delNum := 0 - newNum
		for i := 0; i < delNum; i++ { // scale down
			worker := <-pool.queue
			if worker.numTask == 0 {
				worker.isKilled = true
				pool.cleaningWorkers[worker.workerId] = worker // add to cleaning map
				delete(pool.workers, worker.workerId)
				go pool.DeleteInstance(worker)
			} else {
				pool.queue <- worker // put back to the queue
				i -= 1               // not deleted, i should not change
				time.Sleep(5 * time.Second)
			}
		}
	}
	if pool.cleanedWorker != nil {
		// check the target and running/starting
		if pool.Size() < pool.target {
			pool.startingWorkers[pool.cleanedWorker.workerId] = pool.cleanedWorker
			pool.cleanedWorker = nil
			pool.needRestart = true
		}
		pool.cleanedWorker = nil
	}
	if pool.destroyedWorker != nil {
		if pool.Size() < pool.target {
			// start a new worker
			nextId := pool.nextId
			pool.nextId += 1
			worker := pool.NewWorker(nextId)
			pool.workers[worker.workerId] = worker
			pool.queue <- worker
			pool.startingWorkers[worker.workerId] = worker
			pool.destroyedWorker = nil
			go pool.CreateInstance(worker)
		}
		pool.destroyedWorker = nil
	}
}

//run lambda function
func (pool *WorkerPool) RunLambda(w http.ResponseWriter, r *http.Request) {
	worker := <-pool.queue
	atomic.AddInt32(&worker.numTask, 1)
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
	atomic.AddInt32(&worker.numTask, -1)
}

//return wokers' id and number of outstanding tasks
func (pool *WorkerPool) StatusTasks() map[string]int32 {
	var output = map[string]int32{}

	for workerId, worker := range pool.workers {	
		output[workerId] = worker.numTask
	}
	return output
}

//return status of cluster
func (pool *WorkerPool) StatusCluster() map[string]int {
	var output = map[string]int{}

	output["starting"] = len(pool.startingWorkers)
	output["running"] = len(pool.runningWorkers)
	output["cleaning"] = len(pool.cleaningWorkers)
	output["destroying"] = len(pool.destroyingWorkers)
	
	return output
}

//kill and delete all workers
func (pool *WorkerPool) Close() {
	pool.target = 0
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

package boss

import (
	"fmt"
	"log"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
	"os"
	"os/exec"
	"os/user"
)

type WorkerState int
const (
	STARTING WorkerState = iota
	RUNNING
	CLEANING
	DESTROYING
)

var (
	clusterLogFile	*os.File
	taskLogFile		*os.File
	clusterLog		*log.Logger
	taskLog			*log.Logger
	totalTask		int32
	sumLatency		int64
	nLatency		int64
)

type WorkerPool struct {
	nextId             	int
	target             	int
	workers			   	[]map[string]*Worker
	queue				chan *Worker
	WorkerPoolPlatform
	Scaling
	sync.Mutex
}

//platform specific attributes and functions
type WorkerPoolPlatform interface {
	NewWorker(nextId int) *Worker  //return new worker struct
	CreateInstance(worker *Worker) //create new instance in the cloud platform
	DeleteInstance(worker *Worker) //delete cloud platform instance associated with give worker struct
}

type Worker struct {
	workerId       string
	workerIp       string
	numTask        int32
	WorkerPlatform
	pool           *WorkerPool
	state		   WorkerState  //state as enum
}

type WorkerPlatform interface {
	//platform specific attributes and functions
	//do not require any functions yet
}

func NewWorkerPool() *WorkerPool {
	clusterLogFile, _ = os.Create("cluster.log")
	taskLogFile, _ = os.Create("tasks.log")
	clusterLog = log.New(clusterLogFile, "", 0)
	taskLog = log.New(taskLogFile, "", 0)
	clusterLog.SetFlags(log.LstdFlags)
	taskLog.SetFlags(log.LstdFlags)

	var pool *WorkerPool
	if Conf.Platform == "gcp" {
		pool = NewGcpWorkerPool()
	} else if Conf.Platform == "azure" {
		pool = NewAzureWorkerPool()
	// } else if Conf.Platform == "do" {
	// 	pool = NewDoWorkerPool()
	} else if Conf.Platform == "mock" {
		pool = NewMockWorkerPool()
	}

	pool.nextId = 1
	pool.workers = []map[string]*Worker{
		make(map[string]*Worker),	//starting
		make(map[string]*Worker),	//running
		make(map[string]*Worker),	//cleaning
		make(map[string]*Worker),	//destroying
	}
	pool.queue = make(chan *Worker, Conf.Worker_Cap)

	if Conf.Scaling == "auto" {
		pool.Scaling = &ScalingThreshold{}
		pool.SetTarget(1)
	}

	log.Printf("READY: worker pool of type %s", Conf.Platform)

	//log total outstanding tasks
	go func() {
		for true {
			time.Sleep(time.Second)
			var avgLatency int64 = 0
			if nLatency > 0 {
				avgLatency = sumLatency/nLatency
			}
			taskLog.Printf("tasks=%d, average_latency(ms)=%d", totalTask, avgLatency)
		}
	}()

	return pool
}

//return number of workers in the pool
func (pool *WorkerPool) Size() int {
	size := 0
	for i:=0; i<len(pool.workers); i++ {
		size += len(pool.workers[i])
	}
	return size
}

//renamed Scale() -> SetTarget()
func (pool *WorkerPool) SetTarget(target int) {
	pool.Lock()
	pool.target = target
	clusterLog.Printf("set target=%d", pool.target)
	pool.updateCluster()
	pool.Unlock()
}

// lock should be held before calling this function
// add a new worker to the cluster
func (pool *WorkerPool) startNewWorker() {
	log.Printf("starting new worker\n")
	nextId := pool.nextId
	pool.nextId += 1
	worker := pool.NewWorker(nextId)
	worker.state = STARTING
	pool.workers[STARTING][worker.workerId] = worker
	clusterLog.Printf("%s: starting [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]", 
		worker.workerId, pool.target,
		len(pool.workers[STARTING]),
		len(pool.workers[RUNNING]),
		len(pool.workers[CLEANING]),
		len(pool.workers[DESTROYING]))

	pool.Unlock()
	go func() { // should be able to create multiple instances simultaneously
		pool.CreateInstance(worker) //create new instance

		if Conf.Platform == "azure" {
			worker.WorkerPlatform.(*AzureWorker).startWorker()
		} else {
			worker.runCmd("./ol worker --detach") // start worker
		}

		//change state starting -> running
		pool.Lock()
		defer pool.Unlock()
		
		worker.state = RUNNING
		delete(pool.workers[STARTING], worker.workerId) 
		pool.workers[RUNNING][worker.workerId] = worker

		clusterLog.Printf("%s: running [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]", 
		worker.workerId, pool.target,
		len(pool.workers[STARTING]),
		len(pool.workers[RUNNING]),
		len(pool.workers[CLEANING]),
		len(pool.workers[DESTROYING]))
		pool.queue <- worker
		log.Printf("%s ready\n", worker.workerId)
		
		pool.updateCluster()
	}()
	pool.Lock()
}

// lock should be held before calling this function
// recover cleaning worker 
func (pool *WorkerPool) recoverWorker(worker *Worker) {
	log.Printf("recovering %s\n", worker.workerId)
	worker.state = RUNNING
	delete(pool.workers[CLEANING], worker.workerId) 
	pool.workers[RUNNING][worker.workerId] = worker

	clusterLog.Printf("%s: running [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]", 
		worker.workerId, pool.target,
		len(pool.workers[STARTING]),
		len(pool.workers[RUNNING]),
		len(pool.workers[CLEANING]),
		len(pool.workers[DESTROYING]))

	pool.updateCluster()
}

// lock should be held before calling this function
// clean the worker
func (pool *WorkerPool) cleanWorker(worker *Worker) {
	log.Printf("cleaning %s\n", worker.workerId)
	worker.state = CLEANING
	delete(pool.workers[RUNNING], worker.workerId) 
	pool.workers[CLEANING][worker.workerId] = worker

	clusterLog.Printf("%s: cleaning [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]", 
		worker.workerId, pool.target,
		len(pool.workers[STARTING]),
		len(pool.workers[RUNNING]),
		len(pool.workers[CLEANING]),
		len(pool.workers[DESTROYING]))
	
	pool.updateCluster()
	
	pool.Unlock()
	go func() { 
		for worker.numTask > 0 { //wait until all task is completed
			pool.Lock()
			if _, ok := pool.workers[CLEANING][worker.workerId]; !ok {
				return //stop if the worker is recovered
			}
			pool.Unlock()
			time.Sleep(time.Second)
		}

		pool.Lock()
		defer pool.Unlock()
		pool.detroyWorker(worker)
	}()
	pool.Lock()
}

// lock should be held before calling this function
// destroy a worker from the cluster
func (pool *WorkerPool) detroyWorker(worker *Worker) {
	log.Printf("destroying %s\n", worker.workerId)

	worker.state = DESTROYING
	delete(pool.workers[CLEANING], worker.workerId) 
	pool.workers[DESTROYING][worker.workerId] = worker

	clusterLog.Printf("%s: destroying [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]", 
		worker.workerId, pool.target,
		len(pool.workers[STARTING]),
		len(pool.workers[RUNNING]),
		len(pool.workers[CLEANING]),
		len(pool.workers[DESTROYING]))

	pool.Unlock()
	go func() { // should be able to destroy multiple instances simultaneously
		pool.DeleteInstance(worker) //delete new instance

		// remove from cluster
		pool.Lock()
		defer pool.Unlock()
		delete(pool.workers[DESTROYING], worker.workerId)

		log.Printf("%s destroyed\n", worker.workerId)
		clusterLog.Printf("%s: destroyed [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]", 
			worker.workerId, pool.target,
			len(pool.workers[STARTING]),
			len(pool.workers[RUNNING]),
			len(pool.workers[CLEANING]),
			len(pool.workers[DESTROYING]))
		pool.updateCluster()
	}()
	pool.Lock()
}

// lock should be held before calling this function
// called when worker is been evicted from cleaning or destroying map
func (pool *WorkerPool) updateCluster() {
	scaleSize := pool.target - pool.Size() // scaleSize = target - size of cluster

	if scaleSize >= 0 {
		// create new worker if target is bigger than current cluster size
		// this cluster size includes workers in destroying state
		for i := 0; i < scaleSize; i++ {
			pool.startNewWorker()
		}
	} else {
		// clean workers if target is smaller than current cluster size - cleaning worker - destroying worker
		// ex) if target = 3, and running, cleaning, and destroying  = 1, 2, 1, 1 respectively
		//     then, scaleSize = 3 - 5 = -2. toBeClean = 0 since 2 workers in cleaning and destroying will eventually be destroyed.
		toBeClean := -1*scaleSize - len(pool.workers[CLEANING]) - len(pool.workers[DESTROYING])
		for i:=0; i<toBeClean; i++ { //TODO: policy: clean worker with least tasks
			worker := <- pool.queue
			pool.cleanWorker(worker)
		}
	}

	// recover workers if target - starting worker - running worker > 0
	// ex) if target = 5, and running, cleaning, and destroying  = 1, 2, 1, 1 respectively
	//     then, toBeRecover = 5 - 1 - 2 = 2 and recover 1 cleaning worker since destroying worker cannot be recovered
	toBeRecover := pool.target -len(pool.workers[STARTING]) - len(pool.workers[RUNNING])
	for _, worker := range pool.workers[CLEANING] { 
		if toBeRecover <= 0 { //TODO: policy: recover worker with most tasks
			break
		}
		pool.recoverWorker(worker)
		toBeRecover--
	}
}

//run lambda function
func (pool *WorkerPool) RunLambda(w http.ResponseWriter, r *http.Request) {
	starttime := time.Now()
	if len(pool.workers[STARTING]) + len(pool.workers[RUNNING]) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
		if Conf.Scaling == "manual" {
			_, err := w.Write([]byte("no active worker\n"))
			if err != nil {
				log.Printf("no active worker: %s\n", err.Error())
			}
			return
		}
	}
	
	worker := <- pool.queue
	pool.queue <- worker

	atomic.AddInt32(&worker.numTask, 1)
	atomic.AddInt32(&totalTask, 1)
	if Conf.Scaling == "auto" {
		pool.Scale(pool)
	}
	
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
	atomic.AddInt32(&totalTask, -1)

	latency := time.Since(starttime).Milliseconds()
	
	atomic.AddInt64(&sumLatency, latency)
	atomic.AddInt64(&nLatency, 1)
}

//force kill workers
func (pool *WorkerPool) Close() {
	log.Println("closing worker pool")
	
	pool.Lock()

	pool.target = 0
	var wg sync.WaitGroup
	for i:=0; i<3; i++ {
		for _, worker := range pool.workers[i] {
			delete(pool.workers[i], worker.workerId) 
			pool.workers[DESTROYING][worker.workerId] = worker
			
			pool.Unlock()
			wg.Add(1)
			go func(w *Worker) {
				log.Printf("destroying %s\n", worker.workerId)
				clusterLog.Printf("%s: destroying [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]", 
					worker.workerId, pool.target,
					len(pool.workers[STARTING]),
					len(pool.workers[RUNNING]),
					len(pool.workers[CLEANING]),
					len(pool.workers[DESTROYING]))

				pool.DeleteInstance(w)
				
				pool.Lock()
				defer pool.Unlock()

				delete(pool.workers[DESTROYING], w.workerId) 
				clusterLog.Printf("%s: destroyed [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]", 
					worker.workerId, pool.target,
					len(pool.workers[STARTING]),
					len(pool.workers[RUNNING]),
					len(pool.workers[CLEANING]),
					len(pool.workers[DESTROYING]))
				log.Printf("%s destroyed\n", worker.workerId)
				wg.Done()
			}(worker)
			pool.Lock()
		}
	}
	pool.Unlock()
	wg.Wait()
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

// ssh to worker and run command
func (w *Worker) runCmd(command string) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	cmd := fmt.Sprintf("cd %s; %s", cwd, command)

	tries := 10
	for tries > 0 {
		sshcmd := exec.Command("ssh", user.Username+"@"+w.workerIp, "-o", "StrictHostKeyChecking=no", "-C", cmd)
		stdoutStderr, err := sshcmd.CombinedOutput()
		log.Printf("%s\n", stdoutStderr)
		if err == nil {
			break
		}
		tries -= 1
		if tries == 0 {
			log.Println(sshcmd.String())
			panic(err)
		}
		time.Sleep(5 * time.Second)
	}
}

//return wokers' id and number of tasks
func (pool *WorkerPool) StatusTasks() map[string]int {
	var output = map[string]int{}

	output["task/worker"] = 0
	output["total tasks"] = int(totalTask)
	numWorker := len(pool.workers[RUNNING]) + len(pool.workers[STARTING])
	if numWorker > 0 {
		sumTask := 0
		for _, worker := range pool.workers[RUNNING] {
			sumTask += int(worker.numTask)
		}

		output["task/worker"] = sumTask/numWorker
	}

	for i:=0; i<len(pool.workers); i++ {
		for workerId, worker := range pool.workers[i] {	
			output[workerId] = int(worker.numTask)
		}
	}
	return output
}

//return status of cluster
func (pool *WorkerPool) StatusCluster() map[string]int {
	var output = map[string]int{}

	output["starting"] = len(pool.workers[STARTING])
	output["running"] = len(pool.workers[RUNNING])
	output["cleaning"] = len(pool.workers[CLEANING])
	output["destroying"] = len(pool.workers[DESTROYING])
	
	return output
}
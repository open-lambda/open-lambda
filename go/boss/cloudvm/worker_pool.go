package cloudvm

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"sync/atomic"
	"time"

	"github.com/open-lambda/open-lambda/go/boss/etcd"
)

func NewWorkerPool(platform string, worker_cap int, etcdClient *etcd.Client) (*WorkerPool, error) {
	clusterLogFile, _ := os.Create("cluster.log")
	taskLogFile, _ := os.Create("tasks.log")
	
	// Create simple slog handlers for each log file
	clusterLog := slog.New(slog.NewTextHandler(clusterLogFile, nil))
	taskLog := slog.New(slog.NewTextHandler(taskLogFile, nil))

	var pool *WorkerPool
	switch {
	case platform == "mock":
		pool = NewMockWorkerPool()
	case platform == "gcp":
		pool = NewGcpWorkerPool()
	case platform == "local":
		pool = NewLocalWorkerPool()
	default:
		return nil, fmt.Errorf("invalid cloud platform: %s", platform)
	}

	pool.nextId = 1
	pool.workers = []map[string]*Worker{
		make(map[string]*Worker), // starting
		make(map[string]*Worker), // running
		make(map[string]*Worker), // cleaning
		make(map[string]*Worker), // destroying
	}
	pool.queue = make(chan *Worker, worker_cap)
	pool.clusterLogFile = clusterLogFile
	pool.taskLogFile = taskLogFile
	pool.clusterLog = clusterLog
	pool.taskLog = taskLog
	pool.nLatency = 0
	pool.totalTask = 0
	pool.sumLatency = 0
	pool.platform = platform
	pool.worker_cap = worker_cap

	if etcdClient != nil {
		pool.etcd = etcdClient
		if err := pool.restoreFromEtcd(); err != nil {
			return nil, fmt.Errorf("etcd restore failed: %w", err)
		}
	}

	slog.Info(fmt.Sprintf("READY: worker pool of type %s", platform))

	// log total outstanding tasks
	go func() {
		for true {
			time.Sleep(time.Second)
			var avgLatency int64
			if pool.nLatency > 0 {
				avgLatency = pool.sumLatency / pool.nLatency
			} else {
				avgLatency = 0
			}
			taskLog.Info("task metrics",
				"tasks", pool.totalTask,
				"average_latency(ms)", avgLatency)
		}
	}()

	return pool, nil
}

// return number of workers in the pool
func (pool *WorkerPool) Size() int {
	pool.Lock()
	defer pool.Unlock()
	size := 0
	for i := 0; i < len(pool.workers); i++ {
		size += len(pool.workers[i])
	}
	return size
}

// renamed Scale() -> SetTarget()
func (pool *WorkerPool) SetTarget(target int) {
	pool.Lock()

	pool.target = target
	pool.persistMeta()
	pool.clusterLog.Info("set target", "target", pool.target)

	pool.Unlock()

	pool.updateCluster()
}

func (pool *WorkerPool) GetTarget() int {
	return pool.target
}

func (pool *WorkerPool) GetCap() int {
	return pool.worker_cap
}

// add a new worker to the cluster
func (pool *WorkerPool) startNewWorker() {
	pool.Lock()

	slog.Info("starting new worker")
	nextId := pool.nextId
	pool.nextId += 1
	worker := pool.NewWorker(fmt.Sprintf("worker-%d", nextId))
	worker.state = STARTING
	pool.workers[STARTING][worker.workerId] = worker
	pool.persistWorker(worker)
	pool.persistMeta()
	pool.clusterLog.Info("worker starting",
		"worker_id", worker.workerId,
		"target", pool.target,
		"starting", len(pool.workers[STARTING]),
		"running", len(pool.workers[RUNNING]),
		"cleaning", len(pool.workers[CLEANING]),
		"destroying", len(pool.workers[DESTROYING]))

	pool.Unlock()

	go func() { // should be able to create multiple instances simultaneously
		worker.numTask = 1

		if err := pool.CreateInstance(worker); err != nil {
			slog.Error(fmt.Sprintf("Failed to create instance for worker %s: %v", worker.workerId, err))
			panic(err) // TODO: handle error in better way.
		}

		// change state starting -> running
		pool.Lock()

		worker.state = RUNNING
		delete(pool.workers[STARTING], worker.workerId)
		pool.workers[RUNNING][worker.workerId] = worker
		pool.persistWorker(worker)

		pool.clusterLog.Info("worker running",
			"worker_id", worker.workerId,
			"target", pool.target,
			"starting", len(pool.workers[STARTING]),
			"running", len(pool.workers[RUNNING]),
			"cleaning", len(pool.workers[CLEANING]),
			"destroying", len(pool.workers[DESTROYING]))
		pool.queue <- worker
		slog.Info(fmt.Sprintf("%s ready", worker.workerId))
		worker.numTask = 0

		pool.Unlock()

		pool.updateCluster()
	}()
}

// recover cleaning worker
func (pool *WorkerPool) recoverWorker(worker *Worker) {
	pool.Lock()

	slog.Info(fmt.Sprintf("recovering %s", worker.workerId))
	worker.state = RUNNING
	delete(pool.workers[CLEANING], worker.workerId)
	pool.workers[RUNNING][worker.workerId] = worker
	pool.persistWorker(worker)

	pool.clusterLog.Info("worker running",
		"worker_id", worker.workerId,
		"target", pool.target,
		"starting", len(pool.workers[STARTING]),
		"running", len(pool.workers[RUNNING]),
		"cleaning", len(pool.workers[CLEANING]),
		"destroying", len(pool.workers[DESTROYING]))

	pool.Unlock()

	pool.updateCluster()
}

// clean the worker
func (pool *WorkerPool) cleanWorker(worker *Worker) {
	pool.Lock()

	slog.Info(fmt.Sprintf("cleaning %s", worker.workerId))
	worker.state = CLEANING
	delete(pool.workers[RUNNING], worker.workerId)
	pool.workers[CLEANING][worker.workerId] = worker
	pool.persistWorker(worker)

	pool.clusterLog.Info("worker cleaning",
		"worker_id", worker.workerId,
		"target", pool.target,
		"starting", len(pool.workers[STARTING]),
		"running", len(pool.workers[RUNNING]),
		"cleaning", len(pool.workers[CLEANING]),
		"destroying", len(pool.workers[DESTROYING]))

	pool.Unlock()

	go func(worker *Worker) {
		for worker.numTask > 0 { // wait until all task is completed
			slog.Info("worker cleaning progress", "worker_id", worker.workerId, "num_tasks", worker.numTask)
			pool.Lock()
			if _, ok := pool.workers[CLEANING][worker.workerId]; !ok {
				return // stop if the worker is recovered
			}
			pool.Unlock()
			time.Sleep(time.Second)
		}

		pool.destroyWorker(worker)
	}(worker)
}

// destroy a worker from the cluster
func (pool *WorkerPool) destroyWorker(worker *Worker) {
	pool.Lock()

	worker.state = DESTROYING
	delete(pool.workers[CLEANING], worker.workerId)
	pool.workers[DESTROYING][worker.workerId] = worker
	pool.persistWorker(worker)

	pool.clusterLog.Info("worker destroying",
		"worker_id", worker.workerId,
		"target", pool.target,
		"starting", len(pool.workers[STARTING]),
		"running", len(pool.workers[RUNNING]),
		"cleaning", len(pool.workers[CLEANING]),
		"destroying", len(pool.workers[DESTROYING]))

	pool.Unlock()

	go func() { // should be able to destroy multiple instances simultaneously
		err := pool.DeleteInstance(worker) // delete new instance

		if err != nil {
			slog.Error(fmt.Sprintf("Failed to delete instance for worker %s: %v", worker.workerId, err))
			panic(err) // TODO: handle the error in a better way, retry?
		}

		// remove from cluster
		pool.Lock()

		delete(pool.workers[DESTROYING], worker.workerId)
		pool.evictWorker(worker.workerId)

		slog.Info(fmt.Sprintf("%s destroyed", worker.workerId))
		pool.clusterLog.Info("worker destroyed",
			"worker_id", worker.workerId,
			"target", pool.target,
			"starting", len(pool.workers[STARTING]),
			"running", len(pool.workers[RUNNING]),
			"cleaning", len(pool.workers[CLEANING]),
			"destroying", len(pool.workers[DESTROYING]))
		pool.Unlock()

		pool.updateCluster()
	}()
}

// called when worker is been evicted from cleaning or destroying map
func (pool *WorkerPool) updateCluster() {
	scaleSize := pool.target - pool.Size() // scaleSize = target - size of cluster

	if scaleSize > 0 {
		for i := 0; i < scaleSize; i++ {
			pool.startNewWorker()
		}
		return
	}

	pool.Lock()
	toBeClean := -1*scaleSize - len(pool.workers[CLEANING]) - len(pool.workers[DESTROYING]) - len(pool.workers[STARTING])
	pool.Unlock()

	if toBeClean > 0 {
		for i := 0; i < toBeClean; i++ { // TODO: policy: clean worker with least tasks
			worker := <-pool.queue
			slog.Info("cleaning worker", "worker_id", worker.workerId)
			pool.cleanWorker(worker)
		}

		pool.updateCluster()
		return
	}

	pool.Lock()
	toBeRecover := pool.target - len(pool.workers[STARTING]) - len(pool.workers[RUNNING])
	pool.Unlock()

	if toBeRecover > 0 {
		pool.Lock()
		for _, worker := range pool.workers[CLEANING] {
			if toBeRecover <= 0 { // TODO: policy: recover worker with most tasks
				break
			}
			pool.Unlock()
			pool.recoverWorker(worker)
			pool.Lock()
			toBeRecover--
		}
		pool.Unlock()
	}
}

// run lambda function
func (pool *WorkerPool) RunLambda(w http.ResponseWriter, r *http.Request) {
	starttime := time.Now()
	if len(pool.workers[STARTING])+len(pool.workers[RUNNING]) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
	}

	worker := <-pool.queue
	pool.queue <- worker
	atomic.AddInt32(&worker.numTask, 1)
	atomic.AddInt32(&pool.totalTask, 1)

	err := pool.ForwardTask(w, r, worker)

	if err != nil {
		slog.Error(fmt.Sprintf("Failed to forward the task %s: %v", worker.workerId, err))
		// TODO: handle the error better. retry?
	}

	atomic.AddInt32(&worker.numTask, -1)
	atomic.AddInt32(&pool.totalTask, -1)

	latency := time.Since(starttime).Milliseconds()

	atomic.AddInt64(&pool.sumLatency, latency)
	atomic.AddInt64(&pool.nLatency, 1)
}

// force kill workers
func (pool *WorkerPool) Close() {
	slog.Info("closing worker pool")
	pool.SetTarget(0)

	for {
		pool.Lock()
		worker_num := len(pool.workers[STARTING]) + len(pool.workers[RUNNING]) +
			len(pool.workers[CLEANING]) + len(pool.workers[DESTROYING])
		pool.Unlock()
		if worker_num <= 0 {
			break
		}
	}
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
		sshcmd := exec.Command("ssh", user.Username+"@"+w.host, "-o", "StrictHostKeyChecking=no", "-C", cmd)
		stdoutStderr, err := sshcmd.CombinedOutput()
		slog.Info(fmt.Sprintf("%s", stdoutStderr))
		if err == nil {
			break
		}
		tries -= 1
		if tries == 0 {
			slog.Info(sshcmd.String())
			panic(err)
		}
		time.Sleep(5 * time.Second)
	}
}

// return wokers' id and number of tasks
func (pool *WorkerPool) StatusTasks() map[string]int {
	var output = map[string]int{}

	output["task/worker"] = 0
	output["total tasks"] = int(pool.totalTask)
	numWorker := len(pool.workers[RUNNING]) + len(pool.workers[STARTING])
	if numWorker > 0 {
		sumTask := 0
		for _, worker := range pool.workers[RUNNING] {
			sumTask += int(worker.numTask)
		}

		output["task/worker"] = sumTask / numWorker
	}

	for i := 0; i < len(pool.workers); i++ {
		for workerId, worker := range pool.workers[i] {
			output[workerId] = int(worker.numTask)
		}
	}
	return output
}

// return status of cluster
func (pool *WorkerPool) StatusCluster() map[string]int {
	var output = map[string]int{}

	output["starting"] = len(pool.workers[STARTING])
	output["running"] = len(pool.workers[RUNNING])
	output["cleaning"] = len(pool.workers[CLEANING])
	output["destroying"] = len(pool.workers[DESTROYING])

	return output
}

// forward request to worker
func forwardTaskHelper(w http.ResponseWriter, req *http.Request, workerHost string, workerPort string) error {
	host := fmt.Sprintf("%s:%s", workerHost, workerPort)

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

func (pool *WorkerPool) GetWorker() (*Worker, error) {
	if len(pool.workers[STARTING])+len(pool.workers[RUNNING]) == 0 {
		return nil, fmt.Errorf("no worker available")
	}

	// TODO: replace the channel with simple locking
	worker := <-pool.queue
	pool.queue <- worker

	return worker, nil
}

func GetWorkerAddress(worker *Worker) (string, error) {
	if worker == nil {
		return "", fmt.Errorf("worker is nil")
	}
	if worker.host == "" || worker.port == "" {
		return "", fmt.Errorf("worker address is incomplete")
	}
	return fmt.Sprintf("%s:%s", worker.host, worker.port), nil
}

// writes w's current state to etcd - no-op when etcd is disabled
// called immediately after every in-memory state transition.
func (pool *WorkerPool) persistWorker(w *Worker) {
	if pool.etcd == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rec := etcd.WorkerRecord{
		WorkerId: w.workerId,
		State:    int(w.state),
		Host:     w.host,
		Port:     w.port,
	}
	if err := pool.etcd.PutWorker(ctx, rec); err != nil {
		slog.Error("etcd: failed to persist worker", "worker_id", w.workerId, "err", err)
	}
}

// removes a worker's record from etcd after it is fully destroyed.
func (pool *WorkerPool) evictWorker(workerId string) {
	if pool.etcd == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.etcd.DeleteWorker(ctx, workerId); err != nil {
		slog.Error("etcd: failed to evict worker", "worker_id", workerId, "err", err)
	}
}

// persistMeta writes the pool's nextId and target to etcd.
func (pool *WorkerPool) persistMeta() {
	if pool.etcd == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := pool.etcd.PutPoolMeta(ctx, etcd.PoolMeta{NextId: pool.nextId, Target: pool.target}); err != nil {
		slog.Error("etcd: failed to persist pool meta", "err", err)
	}
}

// restoreFromEtcd reads all worker records from etcd and rebuilds the in-memory
// pool state. Called once during NewWorkerPool when etcd is enabled.
func (pool *WorkerPool) restoreFromEtcd() error {
	ctx := context.Background()

	if meta, ok, err := pool.etcd.GetPoolMeta(ctx); err != nil {
		return err
	} else if ok {
		pool.nextId = meta.NextId
		pool.target = meta.Target
		slog.Info("etcd: restored pool meta", "next_id", pool.nextId, "target", pool.target)
	}

	records, err := pool.etcd.RestoreWorkers(ctx)
	if err != nil {
		return err
	}

	for _, rec := range records {
		w := pool.NewWorker(rec.WorkerId)
		w.host = rec.Host
		w.port = rec.Port
		w.state = WorkerState(rec.State)

		switch w.state {
		case RUNNING:
			pool.workers[RUNNING][w.workerId] = w
			pool.queue <- w
			slog.Info("etcd: restored running worker", "worker_id", w.workerId, "host", w.host, "port", w.port)

		case CLEANING:
			// boss died while worker was draining -> treat as RUNNING;
			// updateCluster() will clean it if target < size.
			w.state = RUNNING
			pool.workers[RUNNING][w.workerId] = w
			pool.queue <- w
			pool.persistWorker(w)
			slog.Info("etcd: restored cleaning worker as running", "worker_id", w.workerId)

		case STARTING, DESTROYING:
			// mid-transition state: actual state is unknown. Drop it and let
			// updateCluster() launch replacements to reach target.
			pool.evictWorker(w.workerId)
			slog.Warn("etcd: dropped ambiguous worker", "worker_id", w.workerId, "state", w.state)
		}
	}

	slog.Info("etcd: pool restore complete", "running", len(pool.workers[RUNNING]), "target", pool.target)
	return nil
}

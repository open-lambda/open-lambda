package cloudvm

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/open-lambda/open-lambda/ol/boss/loadbalancer"
)

func NewWorkerPool(platform string, worker_cap int) (*WorkerPool, error) {
	clusterLogFile, _ := os.Create("cluster.log")
	taskLogFile, _ := os.Create("tasks.log")
	clusterLog := log.New(clusterLogFile, "", 0)
	taskLog := log.New(taskLogFile, "", 0)
	clusterLog.SetFlags(log.Lmicroseconds)
	taskLog.SetFlags(log.Lmicroseconds)

	var pool *WorkerPool
	switch {
	case platform == "mock":
		pool = NewMockWorkerPool()
	case platform == "gcp":
		pool = NewGcpWorkerPool()
	case platform == "azure":
		pool, err = NewAzureWorkerPool()
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New("invalid cloud platform")
	}

	pool.nextId = 1
	pool.workers = []map[string]*Worker{
		make(map[string]*Worker), //starting
		make(map[string]*Worker), //running
		make(map[string]*Worker), //cleaning
		make(map[string]*Worker), //destroying
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
	// TODO: this is hard-coded. Need to set it changable
	pool.numGroup = loadbalancer.NumGroup
	pool.groups = make(map[int]*GroupWorker)
	pool.nextGroup = 0

	pool.taksId = 0

	// This is for traces used to foward tasks
	loadbalancer.Traces = loadbalancer.LoadTrace()
	loadbalancer.Lb = loadbalancer.InitLoadBalancer()

	log.Printf("READY: worker pool of type %s", platform)

	//log total outstanding tasks
	go func() {
		for true {
			time.Sleep(time.Second)
			var avgLatency int64 = 0
			if pool.nLatency > 0 {
				avgLatency = pool.sumLatency / pool.nLatency
			}
			taskLog.Printf("tasks=%d, average_latency(ms)=%d", pool.totalTask, avgLatency)
		}
	}()

	return pool, nil
}

//return number of workers in the pool
func (pool *WorkerPool) Size() int {
	pool.Lock()
	defer pool.Unlock()
	size := 0
	for i := 0; i < len(pool.workers); i++ {
		size += len(pool.workers[i])
	}
	return size
}

//renamed Scale() -> SetTarget()
func (pool *WorkerPool) SetTarget(target int) {
	pool.Lock()

	pool.target = target
	pool.clusterLog.Printf("set target=%d", pool.target)

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

	log.Printf("starting new worker\n")

	nextId := pool.nextId
	pool.nextId += 1
	worker := pool.NewWorker(fmt.Sprintf("worker-%d", nextId))
	logPath := fmt.Sprintf("%s_funcLog.log", worker.workerId)
	funcLogFile, _ := os.Create(logPath)
	funcLog := log.New(funcLogFile, "", 0)
	worker.state = STARTING
	pool.workers[STARTING][worker.workerId] = worker
	pool.clusterLog.Printf("%s: starting [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]",
		worker.workerId, pool.target,
		len(pool.workers[STARTING]),
		len(pool.workers[RUNNING]),
		len(pool.workers[CLEANING]),
		len(pool.workers[DESTROYING]))
	worker.funcLog = funcLog

	pool.Unlock()

	go func() { // should be able to create multiple instances simultaneously
		worker.numTask = 1
		err := pool.CreateInstance(worker) //create new instance
		if err != nil {
			log.Fatalf(err.Error())
		}
		// TODO: need to handle this error, not panic (may use channel?)
		workerIdDigit, err := strconv.Atoi(getAfterSep(worker.workerId, "-"))
		// Assign the worker to the group
		// TODO: need to find the most busy worker and double it
		assignedGroup := workerIdDigit%loadbalancer.NumGroup - 1 // -1 because starts from 0
		if assignedGroup == -1 {
			assignedGroup = loadbalancer.NumGroup - 1
		}
		// fmt.Printf("Debug: %d\n", assignedGroup)
		if pool.platform == "gcp" {
			worker.runCmd("./ol worker up -d") // start worker
		} else if pool.platform == "azure" {
			err = worker.start()
			if err != nil {
				// TODO: Handle error (may use channel?)
				log.Fatalln(err)
			}
		}

		//change state starting -> running
		pool.Lock()

		worker.state = RUNNING
		delete(pool.workers[STARTING], worker.workerId)
		pool.workers[RUNNING][worker.workerId] = worker

		pool.clusterLog.Printf("%s: running [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]",
			worker.workerId, pool.target,
			len(pool.workers[STARTING]),
			len(pool.workers[RUNNING]),
			len(pool.workers[CLEANING]),
			len(pool.workers[DESTROYING]))
		pool.queue <- worker
		log.Printf("%s ready\n", worker.workerId)
		worker.numTask = 0
		// update the worker's assigned group
		worker.groupId = assignedGroup

		// update the group stuff in pool
		if _, ok := pool.groups[assignedGroup]; !ok {
			// this group hasn't been created
			pool.groups[assignedGroup] = &GroupWorker{
				groupId:      pool.nextGroup,
				groupWorkers: make(map[string]*Worker),
			}
			pool.nextGroup += 1
			pool.nextGroup %= loadbalancer.NumGroup - 1
		}
		fmt.Printf("Debug: %d\n", assignedGroup)
		group := pool.groups[assignedGroup]
		group.groupWorkers[worker.workerId] = worker

		pool.Unlock()

		pool.updateCluster()
	}()
}

// recover cleaning worker
func (pool *WorkerPool) recoverWorker(worker *Worker) {
	pool.Lock()

	log.Printf("recovering %s\n", worker.workerId)
	worker.state = RUNNING
	delete(pool.workers[CLEANING], worker.workerId)
	pool.workers[RUNNING][worker.workerId] = worker

	pool.clusterLog.Printf("%s: running [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]",
		worker.workerId, pool.target,
		len(pool.workers[STARTING]),
		len(pool.workers[RUNNING]),
		len(pool.workers[CLEANING]),
		len(pool.workers[DESTROYING]))

	// group stuff
	workerGroup := pool.groups[worker.groupId]
	workerGroup.groupWorkers[worker.workerId] = worker

	pool.updateCluster()
}

// clean the worker
func (pool *WorkerPool) cleanWorker(worker *Worker) {
	pool.Lock()

	log.Printf("cleaning %s\n", worker.workerId)
	worker.state = CLEANING
	delete(pool.workers[RUNNING], worker.workerId)
	pool.workers[CLEANING][worker.workerId] = worker

	pool.clusterLog.Printf("%s: cleaning [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]",
		worker.workerId, pool.target,
		len(pool.workers[STARTING]),
		len(pool.workers[RUNNING]),
		len(pool.workers[CLEANING]),
		len(pool.workers[DESTROYING]))

	// group stuff
	workerGroup := pool.groups[worker.groupId]
	delete(workerGroup.groupWorkers, worker.workerId)

	pool.Unlock()

	go func(worker *Worker) {
		for worker.numTask > 0 { //wait until all task is completed
			fmt.Printf("%s cleaning: %d", worker.workerId, worker.numTask)
			pool.Lock()
			if _, ok := pool.workers[CLEANING][worker.workerId]; !ok {
				return //stop if the worker is recovered
			}
			pool.Unlock()
			time.Sleep(time.Second)
		}

		pool.detroyWorker(worker)
	}(worker)
}

// destroy a worker from the cluster
func (pool *WorkerPool) detroyWorker(worker *Worker) {
	pool.Lock()

	worker.state = DESTROYING
	delete(pool.workers[CLEANING], worker.workerId)
	pool.workers[DESTROYING][worker.workerId] = worker

	pool.clusterLog.Printf("%s: destroying [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]",
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

		delete(pool.workers[DESTROYING], worker.workerId)

		log.Printf("%s destroyed\n", worker.workerId)
		pool.clusterLog.Printf("%s: destroyed [target=%d, starting=%d, running=%d, cleaning=%d, destroying=%d]",
			worker.workerId, pool.target,
			len(pool.workers[STARTING]),
			len(pool.workers[RUNNING]),
			len(pool.workers[CLEANING]),
			len(pool.workers[DESTROYING]))
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
		for i := 0; i < toBeClean; i++ { //TODO: policy: clean worker with least tasks
			worker := <-pool.queue
			fmt.Printf("cleaning %s\n", worker.workerId)
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
			if toBeRecover <= 0 { //TODO: policy: recover worker with most tasks
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

func getAfterSep(str string, sep string) string {
	res := ""
	if idx := strings.LastIndex(str, sep); idx != -1 {
		res = str[idx+1:]
	}
	return res
}

// getURLComponents parses request URL into its "/" delimated components
func getURLComponents(r *http.Request) []string {
	path := r.URL.Path

	// trim prefix
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	// trim trailing "/"
	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	components := strings.Split(path, "/")
	return components
}

func readFirstLine(path string) string {
	file, err := os.Open(path)
	var res string
	if err != nil {
		log.Fatalf("Failed to open file: %s", err)
	}
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		res = scanner.Text() // Outputs the first line
	}

	// Check for errors during scanning
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %s", err)
	}
	defer file.Close()
	return res
}

func isStrExists(str string, list []string) bool {
	exists := false
	for _, s := range list {
		if s == str {
			exists = true
			break
		}
	}
	return exists
}

//run lambda function
func (pool *WorkerPool) RunLambda(w http.ResponseWriter, r *http.Request) {
	pool.Lock()
	pool.taksId += 1
	thisTask := pool.taksId
	pool.Unlock()
	starttime := time.Now()

	assignSuccess := false
	if len(pool.workers[STARTING])+len(pool.workers[RUNNING]) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
	}
	var worker *Worker
	if loadbalancer.Lb.LbType == loadbalancer.Random {
		worker = <-pool.queue
		pool.queue <- worker
	} else {
		// TODO: what if the designated worker isn't up yet?
		// Current solution: then randomly choose one that is up
		// step 1: get its dependencies
		urlParts := getURLComponents(r)
		var pkgs []string
		if len(urlParts) < 2 {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("expected invocation format: /run/<lambda-name>"))
		} else {
			// components represent run[0]/<name_of_sandbox>[1]/<extra_things>...
			// ergo we want [1] for name of sandbox
			urlParts := getURLComponents(r)
			// TODO: if user changes the code, one worker will know that, boss cannot know that. How to handle this?
			if len(urlParts) == 2 {
				img := urlParts[1]
				path := fmt.Sprintf("default-ol/registry/%s.py", img)
				firstLine := readFirstLine(path)
				sub := getAfterSep(firstLine, ":")
				pkgs = strings.Split(sub, ",")
				// get direct packages the function needs
				for i := range pkgs {
					pkgs[i] = strings.TrimSpace(pkgs[i])
				}

			} else {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("expected invocation format: /run/<lambda-name>"))
			}
		}
		// get indirect packages the function needs
		pkgsDeps := pkgs // this is a list of all packages required
		for _, pkg := range pkgs {
			for _, trace := range loadbalancer.Traces.Data {
				if trace.Name == pkg {
					pkgsDeps = append(pkgsDeps, trace.Deps...)
				}
			}
		}
		path := "call_matrix_sample.csv"
		firstLine := readFirstLine(path)
		matrix_pkgs := strings.Split(firstLine, ",")
		matrix_pkgs = matrix_pkgs[1:]
		var vec_matrix []int
		for _, name := range matrix_pkgs {
			if isStrExists(name, pkgsDeps) {
				vec_matrix = append(vec_matrix, 1)
			} else {
				vec_matrix = append(vec_matrix, 0)
			}
		}
		// step 2: get assigned group
		var targetGroup int
		if loadbalancer.Lb.LbType == loadbalancer.KMeans {
			vec_float := make([]float64, len(vec_matrix))
			for i, v := range vec_matrix {
				vec_float[i] = float64(v)
			}
			targetGroup = loadbalancer.KMeansGetGroup(vec_float)
		} else if loadbalancer.Lb.LbType == loadbalancer.KModes {
			targetGroup = loadbalancer.KModesGetGroup(vec_matrix)
		}
		// fmt.Printf("Debug: targetGroup: %d\n", targetGroup)
		// step3: get assigned worker randomly
		assignSuccess = false
		// Might be problem: shoud I add lock here?
		if group, ok := pool.groups[targetGroup]; ok { // exists this group
			// fmt.Println(len(group.groupWorkers))
			if len(group.groupWorkers) > 0 {
				// Seed the random number generator
				rand.Seed(time.Now().UnixNano())
				// Generate a random index
				randIndex := rand.Intn(len(group.groupWorkers))
				for _, thisWorker := range group.groupWorkers {
					if randIndex == 0 {
						assignSuccess = true
						worker = thisWorker
						break
					}
					randIndex--
				}
			}
		}
		// if assign to a worker failed, randomly pick one
		if !assignSuccess {
			fmt.Println("assign to a group (KMeans/KModes) failed")
			worker = <-pool.queue
			pool.queue <- worker
		}
	}
	assignTime := time.Since(starttime).Milliseconds()

	atomic.AddInt32(&worker.numTask, 1)
	atomic.AddInt32(&pool.totalTask, 1)

	pool.ForwardTask(w, r, worker)

	atomic.AddInt32(&worker.numTask, -1)
	atomic.AddInt32(&pool.totalTask, -1)

	latency := time.Since(starttime).Milliseconds()

	pool.Lock()
	if loadbalancer.Lb.LbType == loadbalancer.Random {
		worker.funcLog.Printf("{\"workernum\": %d, \"task\": %d, \"time\": %d, \"assignTime\": %d, \"assign\": \"Random\"}\n", len(pool.workers[RUNNING]), thisTask, latency, assignTime)
	} else {
		if assignSuccess {
			worker.funcLog.Printf("{\"workernum\": %d, \"task\": %d, \"time\": %d, \"assignTime\": %d, \"assign\": \"Success\"}\n", len(pool.workers[RUNNING]), thisTask, latency, assignTime)
		} else {
			worker.funcLog.Printf("{\"workernum\": %d, \"task\": %d, \"time\": %d, \"assignTime\": %d, \"assign\": \"Unsuccess\"}\n", len(pool.workers[RUNNING]), thisTask, latency, assignTime)
		}
	}
	pool.Unlock()

	atomic.AddInt64(&pool.sumLatency, latency)
	atomic.AddInt64(&pool.nLatency, 1)
}

//force kill workers
func (pool *WorkerPool) Close() {
	log.Println("closing worker pool")
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
		var sshcmd *exec.Cmd
		if w.pool.platform == "azure" {
			sshcmd = exec.Command("ssh", "-i", AzureConf.Resource_groups.Rgroup[0].SSHKey, user.Username+"@"+w.workerIp, "-o", "StrictHostKeyChecking=no", "-C", cmd)
		} else if w.pool.platform == "gcp" {
			sshcmd = exec.Command("ssh", user.Username+"@"+w.workerIp, "-o", "StrictHostKeyChecking=no", "-C", cmd)
		}
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

//return status of cluster
func (pool *WorkerPool) StatusCluster() map[string]int {
	var output = map[string]int{}

	output["starting"] = len(pool.workers[STARTING])
	output["running"] = len(pool.workers[RUNNING])
	output["cleaning"] = len(pool.workers[CLEANING])
	output["destroying"] = len(pool.workers[DESTROYING])

	return output
}

// forward request to worker
// TODO: this is kept for other platforms
func forwardTaskHelper(w http.ResponseWriter, req *http.Request, workerIp string) error {
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

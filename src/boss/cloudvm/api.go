package cloudvm

import (
	"log"
	"net/http"
	"os"
	"sync"
)

// The WorkerState integer
type WorkerState int

// Different integers represents different numbers
const (
	STARTING   WorkerState = 0
	RUNNING    WorkerState = 1
	CLEANING   WorkerState = 2 // waiting for already-started requests to complete (so can kill cleanly)
	DESTROYING WorkerState = 3
)

const (
	resourceGroupName = "ol-group"
	location          = "eastus"
	disk              = "ol-boss-new_OsDisk_1_a3f9be95785c437fabe8819c5807ca13"
	vnet              = "ol-boss-new-vnet"
	snapshot          = "ol-boss-new-snapshot"
)

const (
	tree_path    = "/home/azureuser/paper-tree-cache/analysis/17/trials/0/tree-v2.node-320.json"
	test_path    = "/home/azureuser/paper-tree-cache/analysis/17/"
	ssh_key_path = "/home/azureuser/.ssh/ol-boss_key.pem"
)

/*
Defines the interface for platform-specific functions
*/
type WorkerPoolPlatform interface {
	NewWorker(workerId string) *Worker   //return new worker struct
	CreateInstance(worker *Worker) error //create new instance in the cloud platform
	DeleteInstance(worker *Worker) error //delete cloud platform instance associated with give worker struct
	ForwardTask(w http.ResponseWriter, r *http.Request, worker *Worker)
}

/*
Defines the WorkerPool structure. The first field implements the interface
WorkerPoolPlatform.
*/
type WorkerPool struct {
	WorkerPoolPlatform
	platform   string
	worker_cap int
	sync.Mutex
	nextId  int                  // the next new worker's id
	target  int                  // the target number of running+starting workers
	workers []map[string]*Worker // a slice of maps
	// Slice: index maps to a const WorkerState
	// Map: key=worker id (string), value=pointer to worker
	queue chan *Worker // a queue of running workers

	clusterLogFile *os.File
	taskLogFile    *os.File
	clusterLog     *log.Logger
	taskLog        *log.Logger
	totalTask      int32
	sumLatency     int64
	nLatency       int64

	numGroup  int
	nextGroup int
	groups    map[int]*GroupWorker // this mappes the groupId to the GroupWorker

	taksId int32

	workers_queue map[*Worker]chan string // mappes worker to its channel of handling requests
}

type GroupWorker struct {
	groupId      int                // specifies the group name
	groupWorkers map[string]*Worker // what workers does this group have. The worker in this map must be running
}

/*
Defines the Worker structure
*/
type Worker struct {
	workerId string
	workerIp string
	numTask  int32
	allTaks  int32
	pool     *WorkerPool
	state    WorkerState
	groupId  int

	funcLogFile *os.File
	funcLog     *log.Logger
}

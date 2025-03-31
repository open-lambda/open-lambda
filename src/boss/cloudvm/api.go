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

/*
Defines the interface for platform-specific functions
*/
type WorkerPoolPlatform interface {
	NewWorker(workerId string) *Worker // return new worker struct
	CreateInstance(worker *Worker)     // create new instance in the cloud platform
	DeleteInstance(worker *Worker)     // delete cloud platform instance associated with give worker struct
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
}

/*
Defines the Worker structure
*/
type Worker struct {
	workerId string
	numTask  int32
	pool     *WorkerPool
	state    WorkerState
	port     string
	host     string
}

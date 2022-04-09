package boss

import (
	"log"
	"fmt"
	"net/http"
)

type Invocation struct {
	w http.ResponseWriter
	r *http.Request
	Done chan bool
}

func NewInvocation(w http.ResponseWriter, r *http.Request) *Invocation {
	return &Invocation{w: w, r: r, Done: make(chan bool)}
}

type WorkerPool interface {
	// create worker that serves requests from given channel
	Create(reqChan chan *Invocation) Worker
}

type Worker interface {
	// shutdown associated VM
	Cleanup()
}

// WORKER IMPLEMENTATION: MockWorker

type MockWorkerPool struct {
	nextId int
}

type MockWorker struct {
	workerId int
	reqChan chan *Invocation
}

func NewMockWorkerPool() *MockWorkerPool {
	return &MockWorkerPool{
		nextId: 1,
	}
}

func (pool *MockWorkerPool) Create(reqChan chan *Invocation) Worker {
	log.Printf("creating mock worker")
	worker := &MockWorker{
		workerId: pool.nextId,
		reqChan: reqChan,
	}
	pool.nextId += 1
	go worker.task()
	return worker
}

func (worker *MockWorker) task() {
	for {
		req := <- worker.reqChan

		if req == nil {
			// nil request sent from Cleanup
			return
		}

		// respond with dummy message
		// (a real Worker will forward it to the OL worker on a different VM)
		s := fmt.Sprintf("hello from MockWorker %d\n", worker.workerId)
		req.w.WriteHeader(http.StatusOK)
		_, err := req.w.Write([]byte(s))
		if err != nil {
			panic(err)
		}
		req.Done <- true
	}
}

func (worker *MockWorker) Cleanup() {
	worker.reqChan <- nil
}

// WORKER IMPLEMENTATION: GcpWorker (TODO)

// WORKER IMPLEMENTATION: AzureWorker (TODO)

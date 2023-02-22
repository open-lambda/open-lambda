package boss

import (
	"fmt"
	"log"
	"net/http"
	"io"
	"io/ioutil"
	"bytes"
)

// Non-platform specific functions mockWorker implementation

type Invocation struct {
	w    http.ResponseWriter
	r    *http.Request
	Done chan bool
}

func NewInvocation(w http.ResponseWriter, r *http.Request) *Invocation {
	return &Invocation{w: w, r: r, Done: make(chan bool)}
}

type WorkerPool interface {
	CreateWorker(reqChan chan *Invocation) //create new worker
	DeleteWorker(workerId string) //delete worker with worker id
	Status() []string //return list of active workers
	Size() int //return number of active workers
	CloseAll() //Close all workers
}

type Worker interface {
	task()
	Close()
}

// WORKER IMPLEMENTATION: MockWorker

type MockWorkerPool struct {
	nextId 		int
	workers		map[string]Worker
}

type MockWorker struct {
	pool *MockWorkerPool
	workerId string
	reqChan  chan *Invocation
	exitChan chan bool
}

// WORKER IMPLEMENTATION: MockWorker

func NewMockWorkerPool() (*MockWorkerPool, error) {
	return &MockWorkerPool {
		nextId:	1,
		workers: map[string]Worker{},
	}, nil
}

func (pool *MockWorkerPool) CreateWorker(reqChan chan *Invocation) {
	log.Printf("creating mock worker")
	workerId := fmt.Sprintf("worker-%d", pool.nextId)
	worker := &MockWorker{
		pool: pool,
		workerId: workerId,
		reqChan:  reqChan,
		exitChan: make(chan bool), //for exiting task() go routine
	}
	pool.nextId += 1
	go worker.task()
	
	pool.workers[workerId] = worker
}

func (pool *MockWorkerPool) DeleteWorker(workerId string) {
	pool.workers[workerId].Close()
}

func (pool *MockWorkerPool) Status() []string {
	var w = []string{}
	for k, _ := range pool.workers {
		w = append(w, k)
	}
	return w
}

func (pool *MockWorkerPool) Size() int {
	return len(pool.workers)
}

func (pool *MockWorkerPool) CloseAll() {
	for _, w := range pool.workers {
		w.Close() 
	}
}

func (worker *MockWorker) task() {
	for {
		var req *Invocation
		select {
		case <-worker.exitChan: //end go routine if value sent via exitChan
			return
		case req = <-worker.reqChan:
		}

		if req == nil { //naive version scaling down (delete any idle worker)
			worker.Close()
			return
		}

		// respond with dummy message
		// (a real Worker will forward it to the OL worker on a different VM)
		// err = forwardTask(req.w, req.r, "")
		s := fmt.Sprintf("hello from %s\n", worker.workerId)
		req.w.WriteHeader(http.StatusOK)
		_, err := req.w.Write([]byte(s))
		if err != nil {
			panic(err)
		}
		req.Done <- true
	}
}

func (worker *MockWorker) Close() {
	select {
	case worker.exitChan <- true: //end task() go rountine
	default:

	}
	
	//shutdown or remove worker-VM
	log.Printf("closing %s\n", worker.workerId)

	delete(worker.pool.workers, worker.workerId)
}

// forward request to worker
func forwardTask(w http.ResponseWriter, req *http.Request, workerIp string) (error) {
    body, err := ioutil.ReadAll(req.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return err
    }

    req.Body = ioutil.NopCloser(bytes.NewReader(body))
    url := fmt.Sprintf("http://%s:%d%s", workerIp, 5000, req.RequestURI) //TODO: read worker port from Config

    workerReq, err := http.NewRequest(req.Method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	
    workerReq.Header = make(http.Header)
    for h, val := range req.Header {
        workerReq.Header[h] = val
    }

	client := http.Client{}
    resp, err := client.Do(workerReq)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadGateway)
        return err
    }
    defer resp.Body.Close()

    io.Copy(w, resp.Body)

	return nil
}
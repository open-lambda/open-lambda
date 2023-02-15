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
	// create worker that serves requests from given channel
	CreateWorker(reqChan chan *Invocation) Worker
}

type Worker interface {
	task()
	Close()
}

// WORKER IMPLEMENTATION: MockWorker

type MockWorkerPool struct {
	nextId int
}

type MockWorker struct {
	workerId int
	reqChan  chan *Invocation
	exitChan chan bool
}

func (worker *MockWorker) Cleanup() {
	worker.reqChan <- nil
}

// TODO: AzureWorker

type AzureWorkerPool struct {
	nextId int
}

type AzureWorker struct {
	pool *AzureWorkerPool
	workerId int
	reqChan  chan *Invocation
}

func NewMockWorkerPool() (*MockWorkerPool, error) {
	return &MockWorkerPool{
		nextId: 1,
	}, nil
}

// WORKER IMPLEMENTATION: MockWorker
func (pool *MockWorkerPool) CreateWorker(reqChan chan *Invocation) Worker {
	log.Printf("creating mock worker")
	worker := &MockWorker{
		workerId: pool.nextId,
		reqChan:  reqChan,
		exitChan: make(chan bool), //for exiting task() go routine
	}
	pool.nextId += 1
	go worker.task()
	return worker
}

func (worker *MockWorker) task() {
	for {
		req := <-worker.reqChan

		select {
		case <-worker.exitChan: //end go routine if value sent via exitChan
			return
		}

		// respond with dummy message
		// (a real Worker will forward it to the OL worker on a different VM)
		// err = forwardTask(req.w, req.r, "")
		s := fmt.Sprintf("hello from MockWorker %d\n", worker.workerId)
		req.w.WriteHeader(http.StatusOK)
		_, err := req.w.Write([]byte(s))
		if err != nil {
			panic(err)
		}
		req.Done <- true
	}
}

func (worker *MockWorker) Close() {
	worker.exitChan <- true //end task() go rountine
	
	//shutdown or remove VM
	fmt.Printf("closing ol-worker-%d\n", worker.workerId)
}

// forward request to worker
func forwardTask(w http.ResponseWriter, req *http.Request, workerIp string) (error) {
    body, err := ioutil.ReadAll(req.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return err
    }

    req.Body = ioutil.NopCloser(bytes.NewReader(body))
    url := fmt.Sprintf("http://%s:%d%s", workerIp, 5000, req.RequestURI) //TODO: read from Config..?

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
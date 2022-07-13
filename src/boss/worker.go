package boss

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"net/http"
)

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
	reqChan  chan *Invocation
}

func NewMockWorkerPool() (*MockWorkerPool, error) {
	return &MockWorkerPool{
		nextId: 1,
	}, nil
}

func (pool *MockWorkerPool) Create(reqChan chan *Invocation) Worker {
	log.Printf("creating mock worker")
	worker := &MockWorker{
		workerId: pool.nextId,
		reqChan:  reqChan,
	}
	pool.nextId += 1
	go worker.task()
	return worker
}

func (worker *MockWorker) task() {
	for {
		req := <-worker.reqChan

		// TODO: channel is shared -- create separate one for cleanup so we kill the right worker
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

// WORKER IMPLEMENTATION: GcpWorker

type GcpWorkerPool struct {
	nextId int
	client *GCPClient
}

type GcpWorker struct {
	pool *GcpWorkerPool
	workerId int
	reqChan  chan *Invocation
}

func NewGcpWorkerPool() (*GcpWorkerPool, error) {
	fmt.Printf("STEP 0: check SSH setup\n")
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	tmp, err := os.ReadFile(filepath.Join(home, ".ssh", "id_rsa.pub"))
	if err != nil {
		return nil, err
	}
	pub := strings.TrimSpace(string(tmp))

	tmp, err = os.ReadFile(filepath.Join(home, ".ssh", "authorized_keys"))
	if err != nil {
		return nil, err
	}
	authorized := strings.Split(string(tmp), "\n")

	matches := false
	for _, v := range authorized {
		if strings.TrimSpace(v) == pub {
			matches = true
			break
		}
	}

	if !matches {
		return nil, fmt.Errorf("could not find id_rsa.pub in authorized_keys, consider running: cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys ")
	}

	fmt.Printf("STEP 1: get access token\n")
	client, err := NewGCPClient("key.json")
	if err != nil {
		return nil, err
	}

	fmt.Printf("STEP 2: lookup instance from IP address\n")
	instance, err := client.GcpInstanceName()
	if err != nil {
		return nil, err
	}
	fmt.Printf("Instance: %s\n", instance)

	fmt.Printf("STEP 3: take crash-consistent snapshot of instance\n")
	disk := instance // assume GCP disk name is same as instance name
	resp, err := client.Wait(client.GcpSnapshot(disk))
	fmt.Println(resp)
	if err != nil {
		return nil, err
	}

	pool := &GcpWorkerPool{
		nextId: 1,
		client: client,
	}
	return pool, nil
}

func (pool *GcpWorkerPool) Create(reqChan chan *Invocation) Worker {
	log.Printf("creating mock worker")
	worker := &GcpWorker{
		pool: pool,
		workerId: pool.nextId,
		reqChan:  reqChan,
	}
	pool.nextId += 1
	go worker.launch()
	return worker
}

func (worker *GcpWorker) launch() {
	client := worker.pool.client
	fmt.Printf("STEP 4: create new VM from snapshot\n")
	VMName := fmt.Sprintf("ol-worker-%d", worker.workerId)
	resp, err := client.Wait(client.LaunchGCP("test-snap", VMName))
	fmt.Println(resp)
	if err != nil {
		panic(err)
	}

	fmt.Printf("STEP 5: start worker\n")
	err = client.StartRemoteWorker()
	if err != nil {
		panic(err)
	}

	go worker.task()
}

func (worker *GcpWorker) task() {
	for {
		req := <-worker.reqChan

		// TODO: channel is shared -- create separate one for cleanup so we kill the right worker
		if req == nil {
			// nil request sent from Cleanup
			return
		}

		// respond with dummy message
		// (a real Worker will forward it to the OL worker on a different VM)
		s := fmt.Sprintf("hello from GcpWorker %d\n", worker.workerId)
		req.w.WriteHeader(http.StatusOK)
		_, err := req.w.Write([]byte(s))
		if err != nil {
			panic(err)
		}
		req.Done <- true
	}
}

func (worker *GcpWorker) Cleanup() {
	worker.reqChan <- nil
}

// WORKER IMPLEMENTATION: AzureWorker (TODO)

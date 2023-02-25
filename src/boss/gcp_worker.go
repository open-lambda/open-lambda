package boss

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// GcpWorker

type GcpWorkerPool struct {
	nextId	int
	client *GCPClient
	workers map[string]Worker
}

type GcpWorker struct {
	pool *GcpWorkerPool
	workerId string
	workerIp string
	reqChan  chan *Invocation
	exitChan chan bool
}

// // WORKER IMPLEMENTATION: GcpWorker

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

	fmt.Printf("STEP 1a: lookup region and zone from metadata server\n")
	region, zone, err := client.GcpProjectZone()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Region: %s\nZone: %s\n", region, zone)

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

	pool := &GcpWorkerPool {
		nextId:		1,
		client: client,
		workers: map[string]Worker{},
	}
	return pool, nil
}

func (pool *GcpWorkerPool) CreateWorker(reqChan chan *Invocation) {
	log.Printf("creating gcp worker")
	workerId := fmt.Sprintf("ol-worker-%d", pool.nextId)
	worker := &GcpWorker{
		pool: pool,
		workerId: workerId,
		reqChan:  reqChan,
		exitChan: make(chan bool),
	}
	pool.nextId += 1
	go worker.launch()
	go worker.task()
	
	pool.workers[workerId] = worker
}

func (pool *GcpWorkerPool) DeleteWorker(workerId string) {
	pool.workers[workerId].Close()
}

func (pool *GcpWorkerPool) Status() []string {
	var w = []string{}
	for k, _ := range pool.workers {
		w = append(w, k)
	}
	return w
}

func (pool *GcpWorkerPool) Size() int {
	return len(pool.workers)
}

func (pool *GcpWorkerPool) CloseAll() {
	for _, w := range pool.workers {
		w.Close() 
	}
}

func (worker *GcpWorker) launch() {
	client := worker.pool.client
	fmt.Printf("STEP 4: create new VM from snapshot\n")
	resp, err := client.Wait(client.LaunchGcp("test-snap", worker.workerId)) //TODO: load snapshot name from Config
	fmt.Println(resp)
	if err != nil && resp["error"].(map[string]any)["code"] != "409" { //continue if instance already exists error
		fmt.Printf("instance alreay exists!\n")
		client.startGcpInstance(worker.workerId)
	} else if err != nil {
		panic(err)
	}

	fmt.Printf("STEP 5: start worker\n")
	err = client.RunComandWorker(worker.workerId, "./ol worker --detach")
	if err != nil {
		panic(err)
	}

	lookup, err := client.GcpInstancetoIP()
	if err != nil {
		panic(err)
	}
	worker.workerIp = lookup[worker.workerId]
	
	go worker.task()
}

func (worker *GcpWorker) task() {
	for {

		var req *Invocation
		select {
		case <-worker.exitChan: 
			return
		case req = <-worker.reqChan:
		}

		if req == nil {
			worker.Close()
			return
		}

		err = forwardTask(req.w, req.r, worker.workerIp)
		if err != nil {
			panic(err)
		}

		req.Done <- true
	}
}

func (worker *GcpWorker) Close() {
	select {
    case worker.exitChan <- true:
    }
	client := worker.pool.client



	log.Printf("stopping %s\n", worker.workerId)

	err = client.RunComandWorker(worker.workerId, "./ol kill")
	if err != nil {
		panic(err)
	}
	client.stopGcpInstance(worker.workerId)
	// or instances can be kept running but stop worker...?
	//err := StopRemoteWorker(VMName)

	delete(worker.pool.workers, worker.workerId)
}
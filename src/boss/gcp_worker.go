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
	nextId int
	client *GCPClient
}

type GcpWorker struct {
	pool *GcpWorkerPool
	workerId int
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

	pool := &GcpWorkerPool{
		nextId: 1,
		client: client,
	}
	return pool, nil
}

func (pool *GcpWorkerPool) CreateWorker(reqChan chan *Invocation) Worker {
	log.Printf("creating mock worker")
	worker := &GcpWorker{
		pool: pool,
		workerId: pool.nextId,
		reqChan:  reqChan,
		exitChan: make(chan bool),
	}
	pool.nextId += 1
	go worker.launch()
	return worker
}

func (worker *GcpWorker) launch() {
	client := worker.pool.client
	fmt.Printf("STEP 4: create new VM from snapshot\n")
	VMName := fmt.Sprintf("ol-worker-%d", worker.workerId)
	resp, err := client.Wait(client.LaunchGcp("test-snap", VMName)) //TODO: load from Config
	fmt.Println(resp)
	if err != nil && resp["error"].(map[string]any)["code"] != "409" { //continue if instance already exists error
		fmt.Printf("instance alreay exists!\n")
	} else if err != nil {
		panic(err)
	}

	fmt.Printf("STEP 5: start worker\n")
	err = client.StartRemoteWorker(VMName)
	if err != nil {
		panic(err)
	}

	go worker.task()
}

func (worker *GcpWorker) task() {
	for {
		req := <-worker.reqChan

		select {
		case <-worker.exitChan:
			return
		}

		err = forwardTask(req.w, req.r, "")
		if err != nil {
			panic(err)
		}

		req.Done <- true
	}
}

func (worker *GcpWorker) Close() {
	worker.exitChan <- true
}
package boss

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// WORKER IMPLEMENTATION: GcpWorker
type GcpWorkerPool struct {
	client *GcpClient
}

type GcpWorker struct {
	//no additional attributes yet
}

func NewGcpWorkerPool() (*WorkerPool, error) {
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
	client, err := NewGcpClient("key.json")
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
	disk := instance // assume Gcp disk name is same as instance name
	resp, err := client.Wait(client.GcpSnapshot(disk))
	fmt.Println(resp)
	if err != nil {
		return nil, err
	}

	return &WorkerPool {
		nextId:	1,
		workers: map[string]*Worker{},
		queue: make(chan *Worker, Conf.Worker_Cap),
		WorkerPoolPlatform: &GcpWorkerPool {
			client: client,
		},
	}, nil
}

func (pool *GcpWorkerPool) NewWorker(nextId int) *Worker {
	workerId := fmt.Sprintf("ol-worker-%d", nextId)
   return &Worker{
	   workerId: workerId,
	   workerIp: "",
	   isIdle: true,
	   WorkerPlatform: GcpWorker{},
   }
}

func (pool *GcpWorkerPool) CreateInstance(worker *Worker) {
	client := pool.client
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
}

func (pool *GcpWorkerPool) DeleteInstance(worker *Worker) {
	client := pool.client

	log.Printf("deleting gcp worker: %s\n", worker.workerId)

	err := client.RunComandWorker(worker.workerId, "./ol kill")
	if err != nil {
		panic(err)
	}
	client.deleteGcpInstance(worker.workerId)
}
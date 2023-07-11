package cloudvm

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"net/http"
)

type GcpWorkerPool struct {
	client *GcpClient
}

func NewGcpWorkerPool() *WorkerPool {
	fmt.Printf("STEP 0: check SSH setup\n")
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	tmp, err := os.ReadFile(filepath.Join(home, ".ssh", "id_rsa.pub"))
	if err != nil {
		panic(err)
	}
	pub := strings.TrimSpace(string(tmp))

	tmp, err = os.ReadFile(filepath.Join(home, ".ssh", "authorized_keys"))
	if err != nil {
		panic(err)
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
		panic("could not find id_rsa.pub in authorized_keys, consider running: cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys ")
	}

	fmt.Printf("STEP 1: get access token\n")
	client, err := NewGcpClient("key.json")
	if err != nil {
		panic(err)
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
		panic(err)
	}
	fmt.Printf("Instance: %s\n", instance)

	fmt.Printf("STEP 3: take crash-consistent snapshot of instance\n")
	disk := instance // assume Gcp disk name is same as instance name
	resp, err := client.Wait(client.GcpSnapshot(disk))
	fmt.Println(resp)
	if err != nil {
		panic(err)
	}

	return &WorkerPool{
		WorkerPoolPlatform: &GcpWorkerPool{
			client: client,
		},
	}
}

func (pool *GcpWorkerPool) NewWorker(workerId string) *Worker {
	return &Worker{
		workerId:       workerId,
		workerIp:       "",
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

	lookup, err := client.GcpInstancetoIP()
	if err != nil {
		panic(err)
	}

	worker.workerIp = lookup[worker.workerId]
}

func (pool *GcpWorkerPool) DeleteInstance(worker *Worker) {
	client := pool.client

	log.Printf("deleting gcp worker: %s\n", worker.workerId)
	client.Wait(client.deleteGcpInstance(worker.workerId)) //wait until instance is completely deleted
}

func (pool *GcpWorkerPool) ForwardTask(w http.ResponseWriter, r *http.Request, worker *Worker) {
	forwardTaskHelper(w, r, worker.workerIp)
}

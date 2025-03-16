package cloudvm

import (
	"log"
	"net/http"
	"os/exec"
	"path/filepath"

	"github.com/open-lambda/open-lambda/ol/common"
)

// WORKER IMPLEMENTATION: LocalWorker
type LocalWorkerPoolPlatform struct {
	// no platform specific attributes
}

func NewLocalWorkerPool() *WorkerPool {
	return &WorkerPool{
		WorkerPoolPlatform: &LocalWorkerPoolPlatform{},
	}
}

func (_ *LocalWorkerPoolPlatform) NewWorker(workerId string) *Worker {
	return &Worker{
		workerId: workerId,
		workerIp: "",
	}
}

func (_ *LocalWorkerPoolPlatform) CreateInstance(worker *Worker) {
	log.Printf("Creating new local worker: %s\n", worker.workerId)

	// Initialize the worker directory if it doesn't exist
	initCmd := exec.Command("./ol", "worker", "init", "-p", worker.workerId, "-i", "ol-min")
	err := initCmd.Run()
	if err != nil {
		log.Printf("Failed to initialize worker %s: %v\n", worker.workerId, err)
		return
	}

	// Load the template json file from OL directory.
	configPath := filepath.Join(filepath.Dir(common.Conf.Worker_dir), "template.json")

	log.Printf("Current worker config dir trying to read: %s\n", configPath)

	if LoadWorkerConfigTemplate(configPath) != nil {
		log.Fatalf("Failed to load template.json: %v", err)
	}

	// Start the worker in detached mode
	upCmd := exec.Command("./ol", "worker", "up", "-p", worker.workerId, "-i", "ol-min", "-d")
	err = upCmd.Start()
	if err != nil {
		log.Printf("Failed to start worker %s: %v\n", worker.workerId, err)
		return
	}

	log.Printf("Worker %s started on %s\n", worker.workerId, worker.workerIp)
}

func (_ *LocalWorkerPoolPlatform) DeleteInstance(worker *Worker) {
	log.Printf("Deleting local worker: %s\n", worker.workerId)

	// Stop the worker process
	downCmd := exec.Command("./ol", "worker", "down", "-p", worker.workerId)
	err := downCmd.Run()
	if err != nil {
		log.Printf("Failed to stop worker %s: %v\n", worker.workerId, err)
		return
	}

	log.Printf("Worker %s stopped\n", worker.workerId)
}

func (_ *LocalWorkerPoolPlatform) ForwardTask(w http.ResponseWriter, r *http.Request, worker *Worker) {
	forwardTaskHelper(w, r, worker.workerIp)
}

package cloudvm

import (
	"log"
	"net/http"
	"os"
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
		host:     "localhost",
	}
}

func (_ *LocalWorkerPoolPlatform) CreateInstance(worker *Worker) {
	log.Printf("Creating new local worker: %s\n", worker.workerId)

	// Initialize the worker directory if it doesn't exist
	initCmd := exec.Command("./ol", "worker", "init", "-p", worker.workerId, "-i", "ol-min") // TODO fix the "ol-min hardcoding"
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		log.Printf("Failed to initialize worker %s: %v\n", worker.workerId, err)
		return // TODO return the error
	}

	currPath, err := os.Getwd()
	if err != nil {
		log.Printf("failed to get current path: %v", err)
	}

	workerPath := filepath.Join(currPath, worker.workerId)
	templatePath := GetLocalPlatformConfigDefaults().Path_To_Worker_Config_Template

	// Load worker configuration
	if err := LoadWorkerConfigTemplate(templatePath, workerPath); err != nil {
		log.Printf("Failed to load template.json: %v", err)
		return // TODO return the error
	}

	worker.port = common.Conf.Worker_port

	// Start the worker in detached mode
	upCmd := exec.Command("./ol", "worker", "up", "-p", worker.workerId, "-i", "ol-min", "-d") // TODO fix the "ol-min hardcoding"
	upCmd.Stderr = os.Stderr
	if err := upCmd.Start(); err != nil {
		log.Printf("Failed to start worker %s: %v\n", worker.workerId, err)
		return // TODO return the error
	}

	log.Printf("Worker %s started on %s\n", worker.workerId, worker.port)
}

func (_ *LocalWorkerPoolPlatform) DeleteInstance(worker *Worker) {
	log.Printf("Deleting local worker: %s\n", worker.workerId)

	// Stop the worker process
	downCmd := exec.Command("./ol", "worker", "down", "-p", worker.workerId)
	err := downCmd.Run()
	if err != nil {
		log.Printf("Failed to stop worker %s: %v\n", worker.workerId, err)
		return // TODO return the error
	}

	log.Printf("Worker %s stopped\n", worker.workerId)
}

func (_ *LocalWorkerPoolPlatform) ForwardTask(w http.ResponseWriter, r *http.Request, worker *Worker) {
	forwardTaskHelper(w, r, worker.host, worker.port)
}

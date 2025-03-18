package cloudvm

import (
	"fmt"
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
		workerIp: "",
	}
}

func (_ *LocalWorkerPoolPlatform) CreateInstance(worker *Worker) {
	log.Printf("Creating new local worker: %s\n", worker.workerId)

	// Initialize the worker directory if it doesn't exist
	initCmd := exec.Command("./ol", "worker", "init", "-p", worker.workerId, "-i", "ol-min")
	initCmd.Stderr = os.Stderr // Capture stderr
	if err := initCmd.Run(); err != nil {
		log.Printf("Failed to initialize worker %s: %v\n", worker.workerId, err)
		panic(err)
	}

	// Get the executable directory
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get executable directory: %v\n", err)
		panic(err)
	}
	appDir := filepath.Dir(execPath)

	// Load worker configuration
	workerConfigPath := fmt.Sprintf("%s/%s/config.json", appDir, worker.workerId)
	templatePath := filepath.Join(appDir, "template.json")
	if err := LoadWorkerConfigTemplate(templatePath, workerConfigPath); err != nil {
		log.Printf("Failed to load template.json: %v", err)
		panic(err)
	}

	free, err := isPortFree(common.Conf.Worker_port)
	if err != nil {
		log.Printf("Error checking port: %v\n", err)
		panic(err)
	}

	if !free {
		log.Printf("The port %s is in use. Please change the port number in template.json\n", common.Conf.Worker_port)
		panic("port is in use")
	}

	worker.workerIp = fmt.Sprintf("localhost:%s", common.Conf.Worker_port)

	// Start the worker in detached mode
	upCmd := exec.Command("./ol", "worker", "up", "-p", worker.workerId, "-i", "ol-min", "-d")
	upCmd.Stderr = os.Stderr // Capture stderr
	if err := upCmd.Start(); err != nil {
		log.Printf("Failed to start worker %s: %v\n", worker.workerId, err)
		panic(err)
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

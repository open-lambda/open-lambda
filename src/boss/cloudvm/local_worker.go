package cloudvm

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/open-lambda/open-lambda/ol/common"
)

// WORKER IMPLEMENTATION: LocalWorker
type LocalWorkerPoolPlatform struct {
	nextWorkerPort int
	configTemplate *common.Config
	lock           sync.Mutex
}

func NewLocalWorkerPool() *WorkerPool {
	startPort, _ := strconv.Atoi(LocalPlatformConfig.Worker_Starting_Port)

	templatePath := GetLocalPlatformConfigDefaults().Path_To_Worker_Config_Template

	// Create template.json if it doesn't exist
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		defaultTemplateConfig, err := common.LoadDefaultTemplateConfig()
		if err != nil {
			log.Fatalf("failed to load default template config: %v", err)
		}

		if err := common.ExportConfig(defaultTemplateConfig, templatePath); err != nil {
			log.Fatalf("failed to save template.json: %v", err)
		}
	}

	// Load the template and save locally
	cfg, err := common.ReadInConf(templatePath)
	if err != nil {
		log.Fatalf("failed to load template config: %v", err)
	}

	return &WorkerPool{
		WorkerPoolPlatform: &LocalWorkerPoolPlatform{
			nextWorkerPort: startPort,
			configTemplate: cfg,
		},
	}
}

func (_ *LocalWorkerPoolPlatform) NewWorker(workerId string) *Worker {
	return &Worker{
		workerId: workerId,
		host:     "localhost",
	}
}

func (p *LocalWorkerPoolPlatform) CreateInstance(worker *Worker) {
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
	workerPort := p.GetNextWorkerPort()

	// Load worker configuration
	if err := SaveTemplateConfToWorkerDir(p.configTemplate, workerPath, workerPort); err != nil {
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

func (p *LocalWorkerPoolPlatform) GetNextWorkerPort() string {
	p.lock.Lock()
	defer p.lock.Unlock()

	port := p.nextWorkerPort
	p.nextWorkerPort++
	return strconv.Itoa(port)
}

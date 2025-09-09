package cloudvm

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/open-lambda/open-lambda/go/boss/config"
	"github.com/open-lambda/open-lambda/go/common"
)

// WORKER IMPLEMENTATION: LocalWorker
type LocalWorkerPoolPlatform struct {
	configTemplate *common.Config
	// lock protects nextWorkerPort from race conditions caused by concurrent access.
	lock           sync.Mutex
	nextWorkerPort int
}

func NewLocalWorkerPool() *WorkerPool {
	startPort, _ := strconv.Atoi(config.BossConf.Local.Worker_Starting_Port)
	templatePath := config.BossConf.Local.Path_To_Worker_Config_Template

	// Create template.json if it doesn't exist
	if _, err := os.Stat(templatePath); err != nil {
		if os.IsNotExist(err) {
			// Get the worker config struct
			defaultTemplateConfig, err := common.GetDefaultWorkerConfig("")
			if err != nil {
				slog.Error(fmt.Sprintf("failed to load default template config: %v", err))
				os.Exit(1)
			}

			if err := common.SaveConfig(defaultTemplateConfig, templatePath); err != nil {
				slog.Error(fmt.Sprintf("failed to save template.json: %v", err))
				os.Exit(1)
			}
		} else {
			slog.Error(fmt.Sprintf("failed to stat template path: %v", err))
			os.Exit(1)
		}
	}

	// Load the template and save locally
	cfg, err := common.ReadInConfig(templatePath)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to load template config: %v", err))
		os.Exit(1)
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

func (p *LocalWorkerPoolPlatform) CreateInstance(worker *Worker) error {
	slog.Info(fmt.Sprintf("Creating new local worker: %s", worker.workerId))

	// Initialize the worker directory if it doesn't exist
	// TODO fix the "ol-min hardcoding"
	initCmd := exec.Command("./ol", "worker", "init", "-p", worker.workerId, "-i", "ol-min")
	// TODO: both the boss and this subprocess can write to the same stream concurrently, which may interleave their outputs.
	// The boss should capture the output from initCmd and then print it using slog.Info which is lock-protected
	initCmd.Stderr = os.Stderr
	if err := initCmd.Run(); err != nil {
		slog.Error(fmt.Sprintf("Failed to initialize worker %s: %v", worker.workerId, err))
		return err
	}

	currPath, err := os.Getwd()
	if err != nil {
		slog.Error(fmt.Sprintf("failed to get current path: %v", err))
		return err
	}

	workerPath := filepath.Join(currPath, worker.workerId)
	workerPort := p.GetNextWorkerPort()

	// Load worker configuration
	if err := SaveTemplateConfToWorkerDir(p.configTemplate, workerPath, workerPort); err != nil {
		slog.Error(fmt.Sprintf("Failed to load template.json: %v", err))
		return err
	}

	worker.port = workerPort

	// Start the worker in detached mode
	// TODO fix the "ol-min hardcoding"
	upCmd := exec.Command("./ol", "worker", "up", "-p", worker.workerId, "-i", "ol-min", "-d")
	// TODO: both the boss and this subprocess can write to the same stream concurrently, which may interleave their outputs.
	// The boss should capture the output from initCmd and then print it using slog.Info which is lock-protected
	upCmd.Stderr = os.Stderr
	if err := upCmd.Start(); err != nil {
		slog.Error(fmt.Sprintf("Failed to start worker %s: %v", worker.workerId, err))
		return err
	}

	slog.Info(fmt.Sprintf("Worker %s started on %s", worker.workerId, worker.port))

	return nil
}

func (_ *LocalWorkerPoolPlatform) DeleteInstance(worker *Worker) error {
	slog.Info(fmt.Sprintf("Deleting local worker: %s", worker.workerId))

	// Stop the worker process
	downCmd := exec.Command("./ol", "worker", "down", "-p", worker.workerId)
	err := downCmd.Run()
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to stop worker %s: %v", worker.workerId, err))
		return err
	}

	slog.Info(fmt.Sprintf("Worker %s stopped", worker.workerId))

	return nil
}

func (_ *LocalWorkerPoolPlatform) ForwardTask(w http.ResponseWriter, r *http.Request, worker *Worker) error {
	return forwardTaskHelper(w, r, worker.host, worker.port)
}

func (p *LocalWorkerPoolPlatform) GetNextWorkerPort() string {
	p.lock.Lock()
	defer p.lock.Unlock()

	port := p.nextWorkerPort
	p.nextWorkerPort++
	return strconv.Itoa(port)
}

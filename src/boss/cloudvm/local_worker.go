package cloudvm

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"sync"
)

type LocalWorkerPool struct {
	mu      sync.Mutex              // Mutex to protect concurrent access to the map
	workers map[string]*LocalWorker // Map to store workers by workerId
}

type LocalWorker struct {
	workerId string
	cmd      *exec.Cmd
	port     int
}

func NewLocalWorkerPool() *WorkerPool {
	return &WorkerPool{
		WorkerPoolPlatform: &LocalWorkerPool{
			workers: make(map[string]*LocalWorker), // Initialize the map
		},
	}
}

func getAvailablePort() int {
	// Example: Start from port 5000 and increment until an available port is found
	port := 5000
	for {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port
		}
		port++
	}
}

func (pool *LocalWorkerPool) NewWorker(workerId string, command string, args ...string) *LocalWorker {
	pool.mu.Lock()         // Lock the map for concurrent access
	defer pool.mu.Unlock() // Unlock when the function returns

	port := getAvailablePort()

	argsWithPort := append(args, fmt.Sprintf("--port=%d", port))

	// TODO: create a worker script that starts a HTTP server and listen to the port
	cmd := exec.Command(command, argsWithPort...)
	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start worker %s: %v", workerId, err)
	}

	worker := &LocalWorker{
		workerId: workerId,
		cmd:      cmd,
		port:     port,
	}

	pool.workers[workerId] = worker // Add the worker to the map
	fmt.Printf("Worker %s started on port %d\n", workerId, port)
	return worker
}

func (_ *LocalWorkerPool) CreateInstance(worker *LocalWorker) {
	log.Printf("create new instance of local worker: %s\n", worker.workerId)
}

func (pool *LocalWorkerPool) DeleteInstance(workerId string) {
	pool.mu.Lock()         // Lock the map for concurrent access
	defer pool.mu.Unlock() // Unlock when the function returns

	worker, exists := pool.workers[workerId] // Retrieve the worker from the map
	if !exists {
		log.Printf("Worker %s not found", workerId)
		return
	}

	log.Printf("Stopping worker: %s\n", workerId)
	err := worker.cmd.Process.Kill()
	if err != nil {
		log.Printf("Failed to kill worker %s: %v", workerId, err)
	}
	delete(pool.workers, workerId) // Remove the worker from the map
}

func (pool *LocalWorkerPool) ForwardTask(w http.ResponseWriter, r *http.Request, worker *LocalWorker) {
	forwardTaskHelper(w, r, "localhost") // TODO: pass the port.
}

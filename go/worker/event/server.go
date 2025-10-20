package event

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/open-lambda/open-lambda/go/boss/lambdastore"
	"github.com/open-lambda/open-lambda/go/common"
)

const (
	RUN_PATH             = "/run/"
	PID_PATH             = "/pid"
	STATUS_PATH          = "/status"
	STATS_PATH           = "/stats"
	DEBUG_PATH           = "/debug"
	PPROF_MEM_PATH       = "/pprof/mem"
	PPROF_CPU_START_PATH = "/pprof/cpu-start"
	PPROF_CPU_STOP_PATH  = "/pprof/cpu-stop"

	// Registry paths - same as boss
	// GET /registry
	// POST /registry/{name}
	// DELETE /registry/{name}
	// GET /registry/{name} not implemented
	// GET /registry/{name}/config
	REGISTRY_BASE_PATH = "/registry/"
)

var (
	lambdaStore *lambdastore.LambdaStore
)

// cleanable represents a resource that requires cleanup on worker shutdown.
// All server implementations and managers should implement this interface.
type cleanable interface {
	cleanup()
}

// LambdaManager is now a singleton, one per worker. This is because lambda manager
// temporary file storing cpu profiled data
const CPU_TEMP_PATTERN = ".cpu.*.prof"

var cpuTemp *os.File
var lock sync.Mutex

// HandleGetPid returns process ID, useful for making sure we're talking to the expected server
func HandleGetPid(w http.ResponseWriter, _ *http.Request) {
	// TODO re-enable once logging is configurable
	// log.Printf("Received request to %s\n", r.URL.Path)

	wbody := []byte(strconv.Itoa(os.Getpid()) + "\n")
	if _, err := w.Write(wbody); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Status writes "ready" to the response.
func Status(w http.ResponseWriter, _ *http.Request) {
	// TODO re-enable once logging is configurable
	// log.Printf("Received request to %s\n", r.URL.Path)

	if _, err := w.Write([]byte("ready\n")); err != nil {
		slog.Error(fmt.Sprintf("error in Status: %v", err))
	}
}

func Stats(w http.ResponseWriter, _ *http.Request) {
	// log.Printf("Received request to %s\n", r.URL.Path)
	snapshot := common.SnapshotStats()
	b, err := json.MarshalIndent(snapshot, "", "\t")

	if err != nil {
		panic(err)
	}

	w.Write(b)
}

func PprofMem(w http.ResponseWriter, _ *http.Request) {
	runtime.GC()
	w.Header().Add("Content-Type", "application/octet-stream")
	if err := pprof.WriteHeapProfile(w); err != nil {
		slog.Error(fmt.Sprintf("could not write memory profile: %v", err))
		os.Exit(1)
	}
}

func doCpuStart() error {
	lock.Lock()
	defer lock.Unlock()

	// user error: double start (previous profiling not stopped yet)
	if cpuTemp != nil {
		return fmt.Errorf("Already started cpu profiling\n")
	}

	// fresh cpu profiling
	temp, err := os.CreateTemp("", CPU_TEMP_PATTERN)
	if err != nil {
		slog.Error(fmt.Sprintf("could not create the temp file: %v", err))
		return err
	}

	slog.Info(fmt.Sprintf("Created a temp file: %s", temp.Name()))
	cpuTemp = temp

	if err := pprof.StartCPUProfile(temp); err != nil {
		slog.Error(fmt.Sprintf("could not start cpu profile: %v", err))
		return err
	}

	slog.Info("Started cpu profiling")
	return nil
}

// Starts CPU profiling
func PprofCpuStart(w http.ResponseWriter, _ *http.Request) {
	if err := doCpuStart(); err != nil {
		msg := fmt.Sprintf("%v", err)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(msg)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// Stops CPU profiling, writes profiled data to response, and does cleanup
func PprofCpuStop(w http.ResponseWriter, _ *http.Request) {
	lock.Lock()
	defer lock.Unlock()

	// user error: should start cpu profiling first
	if cpuTemp == nil {
		slog.Error("should start cpu profile before stopping it")
		w.WriteHeader(http.StatusBadRequest) // bad request
		return
	}

	// flush profile data to file
	pprof.StopCPUProfile()
	tempFilename := cpuTemp.Name()
	cpuTemp.Close()
	cpuTemp = nil
	defer os.Remove(tempFilename) // deferred cleanup

	// read data from file
	slog.Info(fmt.Sprintf("Reading from %s", tempFilename))
	buffer, err := ioutil.ReadFile(tempFilename)
	if err != nil {
		slog.Error(fmt.Sprintf("could not read from file %s", tempFilename))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// write profiled data to response
	w.Header().Add("Content-Type", "application/octet-stream")
	if _, err := w.Write(buffer); err != nil {
		slog.Error(fmt.Sprintf("error in PprofCpuStop: %v", err))
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// shutdown performs final cleanup tasks before the worker exits.
// It saves stats, CPU profile data, and removes the PID file.
// Returns an exit code (0 for success, 1 for any errors).
func shutdown(pidPath string) int {
	statsPath := filepath.Join(common.Conf.Worker_dir, "stats.json")
	snapshot := common.SnapshotStats()
	rc := 0

	// "cpu-start"ed but have not "cpu-stop"ped before kill
	slog.Info("save buffered profiled data to cpu.buf.prof")
	if cpuTemp != nil {
		pprof.StopCPUProfile()
		filename := cpuTemp.Name()
		cpuTemp.Close()

		in, err := ioutil.ReadFile(filename)
		if err != nil {
			slog.Error(fmt.Sprintf("error: %s", err))
			rc = 1
		} else if err = ioutil.WriteFile("cpu.buf.prof", in, 0644); err != nil {
			slog.Error(fmt.Sprintf("error: %s", err))
			rc = 1
		}

		os.Remove(filename)
	}

	slog.Info(fmt.Sprintf("save stats to %s", statsPath))
	if s, err := json.MarshalIndent(snapshot, "", "\t"); err != nil {
		slog.Error(fmt.Sprintf("error: %s", err))
		rc = 1
	} else if err := ioutil.WriteFile(statsPath, s, 0644); err != nil {
		slog.Error(fmt.Sprintf("error: %s", err))
		rc = 1
	}

	slog.Info(fmt.Sprintf("Remove %s.", pidPath))
	if err := os.Remove(pidPath); err != nil {
		slog.Error(fmt.Sprintf("error: %s", err))
		rc = 1
	}

	slog.Info(fmt.Sprintf("Exiting worker (PID %d)", os.Getpid()))
	return rc
}

// RegistryHandler handles registry requests using boss's LambdaStore
func RegistryHandler(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, REGISTRY_BASE_PATH)

	// GET /registry - list all lambda functions in registry
	if relPath == "" {
		if r.Method == "GET" {
			lambdaStore.ListLambda(w)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.SplitN(relPath, "/", 2)

	// GET /registry/{name}/config
	if len(parts) == 2 && parts[1] == "config" && r.Method == "GET" {
		lambdaStore.RetrieveLambdaConfig(w, r)
		return
	}

	switch r.Method {
	case "POST":
		lambdaStore.UploadLambda(w, r)
	case "DELETE":
		lambdaStore.DeleteLambda(w, r)
	case "GET":
		http.Error(w, "not implemented", http.StatusNotImplemented)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func Main() error {
	pidPath := filepath.Join(common.Conf.Worker_dir, "worker.pid")
	if _, err := os.Stat(pidPath); err == nil {
		return fmt.Errorf("Previous worker may be running, %s already exists", pidPath)
	} else if !os.IsNotExist(err) {
		// we were hoping to get the not-exist error, but got something else unexpected
		return err
	}

	// start with a fresh env
	if err := os.RemoveAll(common.Conf.Worker_dir); err != nil {
		return err
	} else if err := os.MkdirAll(common.Conf.Worker_dir, 0700); err != nil {
		return err
	}

	slog.Info(fmt.Sprintf("Saved PID %d to file %s", os.Getpid(), pidPath))
	if err := ioutil.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		return err
	}

	var cleanables []cleanable
	var httpServer *http.Server
	var shutdownComplete = make(chan int, 1) // Channel to receive exit code

	// Consolidated cleanup - runs on both normal shutdown and error cases
	defer func() {
		slog.Info("Running cleanup")

		// Clean up all resources in order
		for _, c := range cleanables {
			if c != nil {
				c.cleanup()
			}
		}

		// Perform final shutdown tasks and get exit code
		exitCode := shutdown(pidPath)

		// Exit with appropriate code
		os.Exit(exitCode)
	}()

	// Create custom ServeMux instead of using DefaultServeMux
	mux := http.NewServeMux()

	// things shared by all servers
	mux.HandleFunc(PID_PATH, HandleGetPid)
	mux.HandleFunc(STATUS_PATH, Status)
	mux.HandleFunc(STATS_PATH, Stats)
	mux.HandleFunc(PPROF_MEM_PATH, PprofMem)
	mux.HandleFunc(PPROF_CPU_START_PATH, PprofCpuStart)
	mux.HandleFunc(PPROF_CPU_STOP_PATH, PprofCpuStop)

	// Initialize LambdaStore for registry
	slog.Info(fmt.Sprintf("Worker: Initializing LambdaStore with Registry = \"%s\"", common.Conf.Registry))
	var err error
	lambdaStore, err = lambdastore.NewLambdaStore(common.Conf.Registry, nil)
	if err != nil {
		return fmt.Errorf("failed to initialize lambda store at %s: %w", common.Conf.Registry, err)
	}

	// Registry handler
	mux.HandleFunc(REGISTRY_BASE_PATH, RegistryHandler)

	switch common.Conf.Server_mode {
	case "lambda":
		lambdaServer, err := NewLambdaServer()
		if err != nil {
			return err
		}
		cleanables = append(cleanables, lambdaServer)

		// Always create and start Kafka manager alongside lambda server
		kafkaManager, err := NewKafkaManager(lambdaServer.lambdaMgr)
		if err != nil {
			return fmt.Errorf("failed to create Kafka manager: %w", err)
		}
		cleanables = append(cleanables, kafkaManager)
		slog.Info("Created kafka manager")

		// Register Kafka management endpoint
		mux.HandleFunc("/kafka/register/", HandleKafkaRegister(kafkaManager, lambdaStore))
	case "sock":
		sockServer, err := NewSOCKServer()
		if err != nil {
			return err
		}
		cleanables = append(cleanables, sockServer)
	default:
		return fmt.Errorf("unknown Server_mode %s", common.Conf.Server_mode)
	}

	// Start Kafka manager in background goroutine if we're in lambda mode
	if common.Conf.Server_mode == "lambda" {
		// kafkaManager is guaranteed to be in cleanables[1] for lambda mode
		kafkaManager := cleanables[1].(*KafkaManager)
		go func() {
			kafkaManager.StartConsuming()
		}()
	}

	// Create custom HTTP server for graceful shutdown
	port := fmt.Sprintf("%s:%s", common.Conf.Worker_url, common.Conf.Worker_port)
	httpServer = &http.Server{
		Addr:    port,
		Handler: mux,
	}

	// clean up if signal hits us (e.g., from ctrl-C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-c
		slog.Info("Received kill signal, shutting down gracefully")

		// Close the HTTP server, which will cause ListenAndServe to return
		if err := httpServer.Close(); err != nil {
			slog.Error(fmt.Sprintf("Error closing HTTP server: %v", err))
		}

		shutdownComplete <- 0 // Normal shutdown
	}()

	slog.Info("Starting HTTP server", "port", port)
	if common.Conf.Server_mode == "lambda" {
		slog.Info("Kafka manager running in background")
	}

	// This blocks until httpServer.Close() is called or an error occurs
	err = httpServer.ListenAndServe()

	// Check if this was a graceful shutdown or an error
	if err != nil && err != http.ErrServerClosed {
		// Unexpected error (e.g., port collision)
		slog.Error(fmt.Sprintf("HTTP server error: %v", err))
		shutdownComplete <- 1 // Error exit code
	}

	// Wait for shutdown signal if not already received
	select {
	case exitCode := <-shutdownComplete:
		if exitCode != 0 {
			return fmt.Errorf("server exited with error")
		}
	default:
		// Server stopped for other reasons
	}

	return nil
}

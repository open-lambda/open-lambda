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
	"github.com/open-lambda/open-lambda/go/worker/lambda"
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

func shutdown(pidPath string, server cleanable) {
	server.cleanup()
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
	os.Exit(rc)
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

// preloadAllKafkaLambdas scans the registry for all lambdas with Kafka triggers
// and registers them immediately to avoid lazy loading delays
func preloadAllKafkaLambdas(kafkaServer *KafkaServer, lambdaMgr *lambda.LambdaMgr) error {
	if lambdaStore == nil {
		return fmt.Errorf("lambda store not initialized")
	}

	slog.Info("Starting preload of Kafka lambdas from registry")

	// Get all lambda names from the registry
	lambdaNames := lambdaStore.ListAllLambdas()

	loadedCount := 0
	errorCount := 0

	for _, lambdaName := range lambdaNames {
		// Get lambda configuration
		config, err := lambdaStore.GetLambdaConfig(lambdaName)
		if err != nil {
			slog.Warn("Failed to load config for lambda during preload",
				"lambda", lambdaName,
				"error", err)
			errorCount++
			continue
		}

		// Debug: Print the loaded config
		slog.Info("DEBUG: Loaded config for lambda",
			"lambda", lambdaName,
			"config", config)

		// Check if lambda has Kafka triggers
		if config == nil || len(config.Triggers.Kafka) == 0 {
			slog.Info("Skipping the following lambda due to lack of config",
				"lambda", lambdaName)
			continue // Skip lambdas without Kafka triggers
		}

		// Register Kafka triggers immediately
		err = kafkaServer.RegisterLambdaKafkaTriggers(lambdaName, config.Triggers.Kafka)
		if err != nil {
			slog.Error("Failed to preload Kafka triggers for lambda",
				"lambda", lambdaName,
				"triggers", len(config.Triggers.Kafka),
				"error", err)
			errorCount++
			continue
		}

		loadedCount++
		slog.Info("Preloaded Kafka triggers for lambda",
			"lambda", lambdaName,
			"triggers", len(config.Triggers.Kafka))
	}

	slog.Info("Completed Kafka lambda preload",
		"total_lambdas", len(lambdaNames),
		"loaded_with_kafka", loadedCount,
		"errors", errorCount)

	return nil
}

func Main() (err error) {
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

	// things shared by all servers
	http.HandleFunc(PID_PATH, HandleGetPid)
	http.HandleFunc(STATUS_PATH, Status)
	http.HandleFunc(STATS_PATH, Stats)
	http.HandleFunc(PPROF_MEM_PATH, PprofMem)
	http.HandleFunc(PPROF_CPU_START_PATH, PprofCpuStart)
	http.HandleFunc(PPROF_CPU_STOP_PATH, PprofCpuStop)

	// Initialize LambdaStore for registry
	slog.Info(fmt.Sprintf("Worker: Initializing LambdaStore with Registry = \"%s\"", common.Conf.Registry))
	lambdaStore, err = lambdastore.NewLambdaStore(common.Conf.Registry, nil)
	if err != nil {
		os.Remove(pidPath)
		return fmt.Errorf("failed to initialize lambda store at %s: %w", common.Conf.Registry, err)
	}

	// Registry handler
	http.HandleFunc(REGISTRY_BASE_PATH, RegistryHandler)

	var s cleanable
	var kafkaServer *KafkaServer

	switch common.Conf.Server_mode {
	case "lambda":
		s, err = NewLambdaServer()
		if err != nil {
			os.Remove(pidPath)
			return err
		}

		// Always create and start Kafka server alongside lambda server
		lambdaServer := s.(*LambdaServer)
		kafkaServer, err = NewKafkaServer(lambdaServer.lambdaMgr)
		if err != nil {
			os.Remove(pidPath)
			return fmt.Errorf("failed to create Kafka server: %w", err)
		}
		slog.Info("Created kafka server")

		// Connect the Kafka server to the Lambda manager for trigger registration
		lambdaServer.SetKafkaServer(kafkaServer)

		// Preload all Kafka consumers for lambdas in the registry
		if err := preloadAllKafkaLambdas(kafkaServer, lambdaServer.lambdaMgr); err != nil {
			slog.Warn("Failed to preload some Kafka lambdas", "error", err)
			// Don't fail server startup for preload issues
		}
	case "sock":
		s, err = NewSOCKServer()
		if err != nil {
			os.Remove(pidPath)
			return err
		}
	default:
		return fmt.Errorf("unknown Server_mode %s", common.Conf.Server_mode)
	}

	// Start Kafka server in background goroutine if it was created
	if kafkaServer != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("Panic in Kafka server", "error", r)
				}
			}()

			if err := kafkaServer.StartConsuming(); err != nil {
				slog.Error("Kafka server error", "error", err)
			}
		}()
	}

	// clean up if signal hits us (e.g., from ctrl-C)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func() {
		<-c
		slog.Info("Received kill signal, cleaning up.")

		// Clean up Kafka server first
		if kafkaServer != nil {
			kafkaServer.cleanup()
		}

		shutdown(pidPath, s)
	}()

	port := fmt.Sprintf("%s:%s", common.Conf.Worker_url, common.Conf.Worker_port)
	slog.Info("Starting HTTP server", "port", port)
	if kafkaServer != nil {
		slog.Info("Kafka server running in background")
	}

	err = http.ListenAndServe(port, nil)

	// if ListenAndServer returned, there must have been some issue
	// (probably a port collision)
	if kafkaServer != nil {
		kafkaServer.cleanup()
	}
	s.cleanup()
	os.Remove(pidPath)
	slog.Error(err.Error())
	os.Exit(1)
	return nil
}

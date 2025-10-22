package event

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net"
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
	"time"

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

func WriteFinalStats(pidPath string, server cleanable) {
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

	slog.Info(fmt.Sprintf("Printed final stats of worker (PID %d)", os.Getpid()))
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
	udsMux := http.NewServeMux()
	portMux := http.NewServeMux()
	
	// create handlers for servers
	udsMux.HandleFunc(PID_PATH, HandleGetPid)
	portMux.HandleFunc(STATUS_PATH, Status)
	portMux.HandleFunc(STATS_PATH, Stats)
	portMux.HandleFunc(PPROF_MEM_PATH, PprofMem)
	portMux.HandleFunc(PPROF_CPU_START_PATH, PprofCpuStart)
	portMux.HandleFunc(PPROF_CPU_STOP_PATH, PprofCpuStop)

	// Initialize LambdaStore for registry
	slog.Info(fmt.Sprintf("Worker: Initializing LambdaStore with Registry = \"%s\"", common.Conf.Registry))
	lambdaStore, err = lambdastore.NewLambdaStore(common.Conf.Registry, nil)
	if err != nil {
		os.Remove(pidPath)
		return fmt.Errorf("failed to initialize lambda store at %s: %w", common.Conf.Registry, err)
	}

	// Registry handler
	portMux.HandleFunc(REGISTRY_BASE_PATH, RegistryHandler)

	var s cleanable
	switch common.Conf.Server_mode {
	case "lambda":
		s, err = NewLambdaServer()
	case "sock":
		s, err = NewSOCKServer()
	default:
		return fmt.Errorf("unknown Server_mode %s", common.Conf.Server_mode)
	}
	if err != nil {
		os.Remove(pidPath)
		return err
	}

	// sock file is made in worker directory
	sockPath := filepath.Join(common.Conf.Worker_dir, "ol.sock")
	// remove socket on exit
	defer func() { _ = os.Remove(sockPath) }()
	// worker access sock file
	ln, errUDS := net.Listen("unix", sockPath)
	if errUDS != nil {
		return fmt.Errorf("failed to listen on UNIX domain socket %s: %w", sockPath, errUDS)
	}
	if err := os.Chmod(sockPath, 0o600); err != nil {
		_ = ln.Close()
		return fmt.Errorf("chmod UNIX domain socket sock file %s: %w", sockPath, err)
	}

	// error channel to handle server errors
	errorChannel := make(chan error, 1)
	// error channel to handle signals (e.g., from ctrl-C)
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	udsServer := &http.Server{
		Handler:           udsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	
	port := fmt.Sprintf("%s:%s", common.Conf.Worker_url, common.Conf.Worker_port)
	portServer := &http.Server{
		Addr:    port,
		Handler: portMux,
	}

	// start serving on the UNIX domain socket server
	go func() {
		slog.Info("worker listening on UNIX domain socket", "socket", sockPath)
		err := udsServer.Serve(ln)
		if err != nil && err != http.ErrServerClosed {
			errorChannel <- fmt.Errorf("UNIX domain socket server failed %w", err)
		}
	}()

	// start serving on the HTTP Server
	go func() {
		slog.Info("worker listening on TCP", "port", port)
		err := portServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errorChannel <- fmt.Errorf("Port server failed %w", err)
		}
	}()

	// wait for either kill signal or error from server
	var shutdownSignal error
	select {
	// shutdown due to signal
	case shutdownSignal := <- signalChannel:
		slog.Info("Received kill signal")
	// shutdown due to server error
	case shutdownSignal := <- errorChannel:
		slog.Error("Received server error:," shutdownSignal)
	}

	// shutdown Lambda server
	slog.Info("Shutting down Lambda server...")
	s.cleanup()

	// shutdown UNIX domain socket server
	slog.Info("Shutting down UNIX domain socket server...")
	if err := udsServer.Shutdown(ctx); err != nil {
		slog.Error("Error during UNIX domain socket server shutdown", "err", err)
	}

	// shutdown TCP server
	slog.Info("Shutting down TCP server...")
	if err := portServer.Shutdown(ctx); err != nil {
		slog.Error("Error during TCP server shutdown", "err", err)
	}

	WriteFinalStats(pidPath, s)
	slog.Info(fmt.Sprintf("Remove %s.", pidPath))
	if err := os.Remove(pidPath); err != nil {
		slog.Error(fmt.Sprintf("error: %s", err))
	}

	return nil
}

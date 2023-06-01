package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"runtime"
	"runtime/pprof"
  	"sync"

	"github.com/open-lambda/open-lambda/ol/common"
)

const (
	RUN_PATH    = "/run/"
	PID_PATH    = "/pid"
	STATUS_PATH = "/status"
	STATS_PATH  = "/stats"
	DEBUG_PATH  = "/debug"
	PPROF_MEM_PATH  = "/pprof/mem"
	PPROF_CPU_START_PATH = "/pprof/cpu-start"
	PPROF_CPU_STOP_PATH = "/pprof/cpu-stop" 
)

// temporary file storing cpu profiled data
const CPU_TEMP_PATTERN = ".cpu.*.prof"
var cpuTemp *os.File = nil
var lock sync.Mutex

// GetPid returns process ID, useful for making sure we're talking to the expected server
func GetPid(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to %s\n", r.URL.Path)

	wbody := []byte(strconv.Itoa(os.Getpid()) + "\n")
	if _, err := w.Write(wbody); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Status writes "ready" to the response.
func Status(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to %s\n", r.URL.Path)

	if _, err := w.Write([]byte("ready\n")); err != nil {
		log.Printf("error in Status: %v", err)
	}
}

func Stats(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request to %s\n", r.URL.Path)
	snapshot := common.SnapshotStats()
	if b, err := json.MarshalIndent(snapshot, "", "\t"); err != nil {
		panic(err)
	} else {
		w.Write(b)
	}
}

func PprofMem(w http.ResponseWriter, r *http.Request) {
        runtime.GC()
	w.Header().Add("Content-Type", "application/octet-stream")
        if err := pprof.WriteHeapProfile(w); err != nil {
	    	log.Fatal("could not write memory profile: ", err)
        }
}

func DoCpuStart() error {
	lock.Lock()
	defer lock.Unlock()

	// user error: double start (previous profiling not stopped yet)
	if cpuTemp != nil {
		return fmt.Errorf("Already started cpu profiling\n")
	}
	  
	// fresh cpu profiling
	temp, err := os.CreateTemp("", CPU_TEMP_PATTERN)
	if err != nil {
		log.Printf("could not create the temp file: %v", err)
		return err
	}

	log.Printf("Created a temp file: %s", temp.Name())
	cpuTemp = temp

	if err := pprof.StartCPUProfile(temp); err != nil {
		log.Printf("could not start cpu profile: %v", err)
		return err
	}

	log.Printf("Started cpu profiling\n")
	return nil
}

// Starts CPU profiling
func PprofCpuStart(w http.ResponseWriter, r *http.Request) {
	if err := DoCpuStart(); err != nil {
		msg := fmt.Sprintf("%v", err)
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(msg)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

// Stops CPU profiling, writes profiled data to response, and does cleanup
func PprofCpuStop(w http.ResponseWriter, r *http.Request) {
	lock.Lock()
	defer lock.Unlock()

	// user error: should start cpu profiling first
	if cpuTemp == nil {
		log.Printf("should start cpu profile before stopping it\n")
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
	log.Printf("Reading from %s\n", tempFilename)
	buffer, err := ioutil.ReadFile(tempFilename)
	if err != nil {
		log.Printf("could not read from file %s\n", tempFilename)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// write profiled data to response
	w.Header().Add("Content-Type", "application/octet-stream")
	if _, err := w.Write(buffer); err != nil {
		log.Printf("error in PprofCpuStop: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func Main() (err error) {
	var s interface {
		cleanup()
	}

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

	log.Printf("Saved PID %d to file %s", os.Getpid(), pidPath)
	if err := ioutil.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			log.Printf("Remvoing PID file %s", pidPath)
			os.Remove(pidPath)
		}
	}()

	// things shared by all servers
	http.HandleFunc(PID_PATH, GetPid)
	http.HandleFunc(STATUS_PATH, Status)
	http.HandleFunc(STATS_PATH, Stats)
	http.HandleFunc(PPROF_MEM_PATH, PprofMem)
	http.HandleFunc(PPROF_CPU_START_PATH, PprofCpuStart)
	http.HandleFunc(PPROF_CPU_STOP_PATH, PprofCpuStop)

	switch common.Conf.Server_mode {
	case "lambda":
		s, err = NewLambdaServer()
	case "sock":
		s, err = NewSOCKServer()
	default:
		return fmt.Errorf("unknown Server_mode %s", common.Conf.Server_mode)
	}

	if err != nil {
		return err
	}

	// clean up if signal hits us
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func() {
		<-c
		log.Printf("received kill signal, cleaning up")
		s.cleanup()


		statsPath := filepath.Join(common.Conf.Worker_dir, "stats.json")
		snapshot := common.SnapshotStats()
		rc := 0

		// "cpu-start"ed but have not "cpu-stop"ped before kill
		log.Printf("save buffered profiled data to cpu.buf.prof\n")
		if cpuTemp != nil {
			pprof.StopCPUProfile()
			filename := cpuTemp.Name()
			cpuTemp.Close()

			in, err := ioutil.ReadFile(filename)
			if err != nil {
				log.Printf("error: %s", err)
				rc = 1
			} else if err = ioutil.WriteFile("cpu.buf.prof", in, 0644); err != nil{
				log.Printf("error: %s", err)
				rc = 1
			}

			os.Remove(filename)
		}

		log.Printf("save stats to %s", statsPath)
		if s, err := json.MarshalIndent(snapshot, "", "\t"); err != nil {
			log.Printf("error: %s", err)
			rc = 1
		} else if err := ioutil.WriteFile(statsPath, s, 0644); err != nil {
			log.Printf("error: %s", err)
			rc = 1
		}

		log.Printf("remove worker.pid")
		if err := os.Remove(pidPath); err != nil {
			log.Printf("error: %s", err)
			rc = 1
		}

		log.Printf("exiting worker, PID=%d", os.Getpid())
		os.Exit(rc)
	}()

	port := fmt.Sprintf("%s:%s", common.Conf.Worker_url, common.Conf.Worker_port)
	log.Fatal(http.ListenAndServe(port, nil))
	panic("ListenAndServe should never return")
}

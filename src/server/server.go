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

	"github.com/open-lambda/open-lambda/ol/common"
)

const (
	RUN_PATH    = "/run/"
	PID_PATH    = "/pid"
	STATUS_PATH = "/status"
	STATS_PATH  = "/stats"
	DEBUG_PATH  = "/debug"
)

// GetPid returns process ID, useful for making sure we're talking to the expected server
func GetPid(w http.ResponseWriter, r *http.Request) {
	log.Printf("Receive request to %s\n", r.URL.Path)

	wbody := []byte(strconv.Itoa(os.Getpid()) + "\n")
	if _, err := w.Write(wbody); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Status writes "ready" to the response.
func Status(w http.ResponseWriter, r *http.Request) {
	log.Printf("Receive request to %s\n", r.URL.Path)

	if _, err := w.Write([]byte("ready\n")); err != nil {
		log.Printf("error in Status: %v", err)
	}
}

func Stats(w http.ResponseWriter, r *http.Request) {
	log.Printf("Receive request to %s\n", r.URL.Path)
	snapshot := common.SnapshotStats()
	if b, err := json.MarshalIndent(snapshot, "", "\t"); err != nil {
		panic(err)
	} else {
		w.Write(b)
	}
}

func Main() (err error) {
	var s interface {
		cleanup()
	}

	pidPath := filepath.Join(common.Conf.Worker_dir, "worker.pid")
	if _, err := os.Stat(pidPath); err == nil {
		return fmt.Errorf("previous worker may be running, %s already exists", pidPath)
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

	log.Printf("save PID %d to file %s", os.Getpid(), pidPath)
	if err := ioutil.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0644); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			log.Printf("remove PID file %s", pidPath)
			os.Remove(pidPath)
		}
	}()

	// things shared by all servers
	http.HandleFunc(PID_PATH, GetPid)
	http.HandleFunc(STATUS_PATH, Status)
	http.HandleFunc(STATS_PATH, Stats)

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
		log.Printf("save stats to %s", statsPath)
		if s, err := json.MarshalIndent(snapshot, "", "\t"); err != nil {
			log.Printf("error: %s", err)
		} else if err := ioutil.WriteFile(statsPath, s, 0644); err != nil {
			log.Printf("error: %s", err)
		}

		log.Printf("remove worker.pid")
		os.Remove(pidPath)

		log.Printf("exiting")
		os.Exit(1)
	}()

	port := fmt.Sprintf(":%s", common.Conf.Worker_port)
	log.Fatal(http.ListenAndServe(port, nil))
	panic("ListenAndServe should never return")
}

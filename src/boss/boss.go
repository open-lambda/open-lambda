package boss

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
)

const (
	RUN_PATH         = "/run/"
	BOSS_STATUS_PATH = "/status"
	SCALING_PATH     = "/scaling/worker_count"
	STORAGE_PATH     = "/registry/upload"
	DOWNLOAD_PATH    = "/registry/download"
	DELETE_PATH      = "registry/delete"
)

type Boss struct {
	reqChan chan *Invocation

	mutex      sync.Mutex
	workerPool WorkerPool
	workers    []Worker
}

var m = map[string][]int{"workers": []int{}}

func (b *Boss) BossStatus(w http.ResponseWriter, r *http.Request) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	log.Printf("Receive request to %s\n", r.URL.Path)
	if b, err := json.MarshalIndent(m, "", "\t"); err != nil {
		panic(err)
	} else {
		w.Write(b)
	}
}

func (b *Boss) ScalingWorker(w http.ResponseWriter, r *http.Request) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// STEP 1: get int (worker count) from POST body, or return an error
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, err := w.Write([]byte("POST a count to /scaling/worker_count\n"))
		if err != nil {
			log.Printf("(1) could not write web response: %s\n", err.Error())
		}
		return
	}

	contents, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("could not read body of web request\n"))
		if err != nil {
			log.Printf("(2) could not write web response: %s\n", err.Error())
		}
		return
	}

	worker_count, err := strconv.Atoi(string(contents))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("body of post to /scaling/worker_count should be an int\n"))
		if err != nil {
			log.Printf("(3) could not write web response: %s\n", err.Error())
		}
		return
	}

	if worker_count > 3 {
		worker_count = 3
		// TODO: config cap
		log.Printf("capping workers at %d to avoid big bills during debugging\n", worker_count)
	}

	// STEP 2: adjust worker count (TODO)
	for len(b.workers) < worker_count {
		worker := b.workerPool.Create(b.reqChan)
		b.workers = append(b.workers, worker)
	}

	// TODO: cleanup workers if necessary

	// TODO: respond based on number of workers
	log.Printf("Receive request to %s, worker_count of %d requested\n", r.URL.Path, worker_count)
	var s []int
	m["workers"] = append(s, 1)
	for k, v := range m {
		fmt.Println(k, "value is", v)
	}
	if b, err := json.MarshalIndent(m, "", "\t"); err != nil {
		panic(err)
	} else {
		w.Write(b)
	}
}

func (b *Boss) RunLambda(w http.ResponseWriter, r *http.Request) {
	invocation := NewInvocation(w, r)

	select {
	case b.reqChan <- invocation:
		<-invocation.Done // wait for completion
	default:
		// the channel is not taking requests, must be too busy
		w.WriteHeader(http.StatusTooManyRequests)
		_, err := w.Write([]byte("request queue is full\n"))
		if err != nil {
			log.Printf("(4) could not write web response: %s\n", err.Error())
		}
	}
}

func (b *Boss) StorageLambda(w http.ResponseWriter, r *http.Request) {
	contents, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("could not read body of web request\n"))
		if err != nil {
			log.Printf("(2) could not write web response: %s\n", err.Error())
		}
		return
	}
	Create(string(contents))
}

func (*Boss) DownloadLambda(w http.ResponseWriter, r *http.Request) {
	// contents, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("could not read body of web request\n"))
		if err != nil {
			log.Printf("(2) could not write web response: %s\n", err.Error())
		}
		return
	}
	Download()
}

func (*Boss) DeleteLambda(w http.ResponseWriter, r *http.Request) {
	// contents, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("could not read body of web request\n"))
		if err != nil {
			log.Printf("(2) could not write web response: %s\n", err.Error())
		}
		return
	}
	Delete()
}

func BossMain() (err error) {
	fmt.Printf("WARNING!  Boss incomplete (only use this as part of development process).")

	var pool WorkerPool
	if Conf.Platform == "gcp" {
		pool, err = NewGcpWorkerPool()
	} else if Conf.Platform == "mock" {
		pool, err = NewMockWorkerPool()
	} else {
		return fmt.Errorf("worker pool '%s' not valid", Conf.Platform)
	}
	if err != nil {
		return err
	}
	fmt.Printf("READY: worker pool of type %s", Conf.Platform)

	boss := Boss{
		workerPool: pool,
		reqChan:    make(chan *Invocation),
	}

	// things shared by all servers
	http.HandleFunc(BOSS_STATUS_PATH, boss.BossStatus)
	http.HandleFunc(SCALING_PATH, boss.ScalingWorker)
	http.HandleFunc(RUN_PATH, boss.RunLambda)
	http.HandleFunc(STORAGE_PATH, boss.StorageLambda)
	http.HandleFunc(DOWNLOAD_PATH, boss.DownloadLambda)
	http.HandleFunc(DELETE_PATH, boss.DeleteLambda)

	port := fmt.Sprintf(":%s", Conf.Boss_port)
	fmt.Printf("Listen on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
	panic("ListenAndServe should never return")
}

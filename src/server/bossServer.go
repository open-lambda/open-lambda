package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"io"
	"strconv"

	"github.com/open-lambda/open-lambda/ol/common"
)

const (
	BOSS_STATUS_PATH = "/bstatus"
	SCALING_PATH     = "/scaling/worker_count"
)

type Boss struct {
	mutex      sync.Mutex
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

	// STEP 2: adjust worker count (TODO)

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

func BossMain() (err error) {
	boss := Boss{}

	// things shared by all servers
	http.HandleFunc(BOSS_STATUS_PATH, boss.BossStatus)
	http.HandleFunc(SCALING_PATH, boss.ScalingWorker)

	port := fmt.Sprintf(":%s", common.Conf.Worker_port)
	fmt.Printf("Listen on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
	panic("ListenAndServe should never return")
}

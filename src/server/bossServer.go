package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

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

	log.Printf("Receive request to %s\n", r.URL.Path)
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

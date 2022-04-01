package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/open-lambda/open-lambda/ol/common"
)

const (
	BOSS_STATUS_PATH = "/bstatus"
	SCALING_PATH     = "/scaling/worker_count"
)

var m = map[string][]int{"workers": []int{}}

func BossStatus(w http.ResponseWriter, r *http.Request) {
	log.Printf("Receive request to %s\n", r.URL.Path)
	if b, err := json.MarshalIndent(m, "", "\t"); err != nil {
		panic(err)
	} else {
		w.Write(b)
	}
}

func ScalingWorker(w http.ResponseWriter, r *http.Request) {
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

	// things shared by all servers
	http.HandleFunc(BOSS_STATUS_PATH, BossStatus)
	http.HandleFunc(SCALING_PATH, ScalingWorker)

	port := fmt.Sprintf(":%s", common.Conf.Worker_port)
	log.Fatal(http.ListenAndServe(port, nil))
	panic("ListenAndServe should never return")
}

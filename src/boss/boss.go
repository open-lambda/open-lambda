package boss

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/open-lambda/open-lambda/ol/boss/autoscaling"
	"github.com/open-lambda/open-lambda/ol/boss/cloudvm"
)

const (
	RUN_PATH         = "/run/"
	BOSS_STATUS_PATH = "/status"
	SCALING_PATH     = "/scaling/worker_count"
	SHUTDOWN_PATH    = "/shutdown"
	RESTART_PATH     = "/restart"
	CHANGE_LB_PATH   = "/change_lb"
	CHANGE_TREE_PATH = "/change_tree"
)

type Boss struct {
	workerPool *cloudvm.WorkerPool
	autoScaler autoscaling.Scaling
}

func (b *Boss) BossStatus(w http.ResponseWriter, r *http.Request) {
	log.Printf("Receive request to %s\n", r.URL.Path)

	output := struct {
		State map[string]int `json:"state"`
		Tasks map[string]int `json:"tasks"`
	}{
		b.workerPool.StatusCluster(),
		b.workerPool.StatusTasks(),
	}

	if b, err := json.MarshalIndent(output, "", "\t"); err != nil {
		panic(err)
	} else {
		w.Write(b)
	}

}

func (b *Boss) Close(w http.ResponseWriter, r *http.Request) {
	b.workerPool.Close()
	if Conf.Scaling == "threshold-scaler" {
		b.autoScaler.Close()
	}
	os.Exit(0)
}

func (b *Boss) ScalingWorker(w http.ResponseWriter, r *http.Request) {
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

	if worker_count > Conf.Worker_Cap {
		worker_count = Conf.Worker_Cap
		log.Printf("capping workers at %d to avoid big bills during debugging\n", worker_count)
	}
	log.Printf("Receive request to %s, worker_count of %d requested\n", r.URL.Path, worker_count)

	// STEP 2: adjust target worker count
	b.workerPool.SetTarget(worker_count)

	//respond with status
	b.BossStatus(w, r)
}

func (b *Boss) ChangeTree(w http.ResponseWriter, r *http.Request) {
	// STEP 1: get int (worker count) from POST body, or return an error
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, err := w.Write([]byte("POST a policy to /change_lb\n"))
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

	new_tree := string(contents)
	Conf.Tree_path = new_tree

	b.workerPool.ChangeTree(new_tree)
}

func (b *Boss) RestartWorkers(w http.ResponseWriter, r *http.Request) {
	b.workerPool.Restart()
	b.BossStatus(w, r)
}

func (b *Boss) ChangeLb(w http.ResponseWriter, r *http.Request) {
	// STEP 1: get int (worker count) from POST body, or return an error
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, err := w.Write([]byte("POST a policy to /change_lb\n"))
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

	new_policy := string(contents)
	Conf.Lb = new_policy
	err = checkConf()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err := w.Write([]byte("body of post to /change_policy should be random, sharding, kmeans, kmodes, hash\n"))
		if err != nil {
			log.Printf("(3) could not write web response: %s\n", err.Error())
		}
		return
	}

	b.workerPool.ChangePolicy(new_policy)
}

func BossMain() (err error) {
	fmt.Printf("WARNING!  Boss incomplete (only use this as part of development process).\n")

	pool, err := cloudvm.NewWorkerPool(Conf.Platform, Conf.Worker_Cap)
	if err != nil {
		return err
	}

	boss := Boss{
		workerPool: pool,
	}

	if Conf.Scaling == "threshold-scaler" {
		boss.autoScaler = &autoscaling.ThresholdScaling{}
		boss.autoScaler.Launch(boss.workerPool)
	}

	http.HandleFunc(BOSS_STATUS_PATH, boss.BossStatus)
	http.HandleFunc(SCALING_PATH, boss.ScalingWorker)
	http.HandleFunc(RUN_PATH, boss.workerPool.RunLambda)
	http.HandleFunc(SHUTDOWN_PATH, boss.Close)
	http.HandleFunc(RESTART_PATH, boss.RestartWorkers)
	http.HandleFunc(CHANGE_LB_PATH, boss.ChangeLb)
	http.HandleFunc(CHANGE_TREE_PATH, boss.ChangeTree)

	// clean up if signal hits us
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	signal.Notify(c, os.Interrupt, syscall.SIGINT)
	go func() {
		<-c
		log.Printf("received kill signal, cleaning up")
		boss.Close(nil, nil)
		os.Exit(0)
	}()

	port := fmt.Sprintf(":%s", Conf.Boss_port)
	fmt.Printf("Listen on port %s\n", port)
	return http.ListenAndServe(port, nil) // should never return if successful
}

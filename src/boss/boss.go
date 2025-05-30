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
	"strings"
	"syscall"

	"github.com/open-lambda/open-lambda/ol/boss/autoscaling"
	"github.com/open-lambda/open-lambda/ol/boss/cloudvm"
	"github.com/open-lambda/open-lambda/ol/boss/config"
	"github.com/open-lambda/open-lambda/ol/boss/lambdastore"
)

const (
	RUN_PATH         = "/run/"
	BOSS_STATUS_PATH = "/status"
	SCALING_PATH     = "/scaling/worker_count"
	SHUTDOWN_PATH    = "/shutdown"

	// GET /registry
	// POST /registry/{name}
	// DELETE /registry/{name}
	// GET /registry/{name} not implemented
	// GET /registry/{name}/config
	REGISTRY_BASE_PATH = "/registry/"
)

type Boss struct {
	workerPool  *cloudvm.WorkerPool
	autoScaler  autoscaling.Scaling
	lambdaStore *lambdastore.LambdaStore
}

// BossStatus handles the request to get the status of the boss.
func (boss *Boss) BossStatus(w http.ResponseWriter, req *http.Request) {
	log.Printf("Receive request to %s\n", req.URL.Path)

	output := struct {
		State map[string]int `json:"state"`
		Tasks map[string]int `json:"tasks"`
	}{
		boss.workerPool.StatusCluster(),
		boss.workerPool.StatusTasks(),
	}

	b, err := json.MarshalIndent(output, "", "\t")
	if err != nil {
		panic(err)
	}

	w.Write(b)
}

// Close handles the request to close the boss.
func (b *Boss) Close(_ http.ResponseWriter, _ *http.Request) {
	b.workerPool.Close()
	if config.BossConf.Scaling == "threshold-scaler" {
		b.autoScaler.Close()
	}
}

// ScalingWorker handles the request to scale the number of workers.
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

	if worker_count > config.BossConf.Worker_Cap {
		worker_count = config.BossConf.Worker_Cap
		log.Printf("capping workers at %d to avoid big bills during debugging\n", worker_count)
	}
	log.Printf("Receive request to %s, worker_count of %d requested\n", r.URL.Path, worker_count)

	// STEP 2: adjust target worker count
	b.workerPool.SetTarget(worker_count)

	// respond with status
	b.BossStatus(w, r)
}

func (b *Boss) RegistryHandler(w http.ResponseWriter, r *http.Request) {
	relPath := strings.TrimPrefix(r.URL.Path, REGISTRY_BASE_PATH)

	// GET /registry - list all lambda functions in registry
	if relPath == "" {
		if r.Method == "GET" {
			b.lambdaStore.ListLambda(w, r)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.SplitN(relPath, "/", 2)

	// GET /registry/{name}/config
	if len(parts) == 2 && parts[1] == "config" && r.Method == "GET" {
		b.lambdaStore.GetLambdaConfig(w, r)
		return
	}

	switch r.Method {
	case "POST":
		b.lambdaStore.UploadLambda(w, r)
	case "DELETE":
		b.lambdaStore.DeleteLambda(w, r)
	case "GET":
		http.Error(w, "not implemented", http.StatusNotImplemented)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// BossMain is the main function for the boss.
func BossMain() (err error) {
	fmt.Printf("WARNING!  Boss incomplete (only use this as part of development process).\n")

	pool, err := cloudvm.NewWorkerPool(config.BossConf.Platform, config.BossConf.Worker_Cap)
	if err != nil {
		return err
	}

	store, err := lambdastore.NewLambdaStore(config.BossConf.Lambda_Store_Path, pool)
	if err != nil {
		return err
	}

	boss := Boss{
		workerPool:  pool,
		lambdaStore: store,
	}

	if config.BossConf.Scaling == "threshold-scaler" {
		boss.autoScaler = &autoscaling.ThresholdScaling{}
		boss.autoScaler.Launch(boss.workerPool)
	}

	http.HandleFunc(BOSS_STATUS_PATH, boss.BossStatus)
	http.HandleFunc(SCALING_PATH, boss.ScalingWorker)
	http.HandleFunc(RUN_PATH, boss.workerPool.RunLambda)
	http.HandleFunc(SHUTDOWN_PATH, boss.Close)

	http.HandleFunc(REGISTRY_BASE_PATH, boss.RegistryHandler)

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

	port := fmt.Sprintf(":%s", config.BossConf.Boss_port)
	fmt.Printf("Listen on port %s\n", port)
	return http.ListenAndServe(port, nil) // should never return if successful
}

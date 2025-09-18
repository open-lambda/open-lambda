package cloudvm

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"log"

	"github.com/open-lambda/open-lambda/go/boss/config"
	"github.com/open-lambda/open-lambda/go/common"
)

type GcpWorkerPool struct {
	client *GcpClient
}

func NewGcpWorkerPool() *WorkerPool {
	fmt.Printf("STEP 0: check SSH setup\n")
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	tmp, err := os.ReadFile(filepath.Join(home, ".ssh", "id_rsa.pub"))
	if err != nil {
		panic(err)
	}
	pub := strings.TrimSpace(string(tmp))

	tmp, err = os.ReadFile(filepath.Join(home, ".ssh", "authorized_keys"))
	if err != nil {
		panic(err)
	}
	authorized := strings.Split(string(tmp), "\n")

	matches := false
	for _, v := range authorized {
		if strings.TrimSpace(v) == pub {
			matches = true
			break
		}
	}

	if !matches {
		panic("could not find id_rsa.pub in authorized_keys, consider running: cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys ")
	}

	fmt.Printf("STEP 1: get access token\n")
	client, err := NewGcpClient("key.json")
	if err != nil {
		panic(err)
	}

	fmt.Printf("STEP 1a: lookup region and zone from metadata server\n")
	region, zone, err := client.GcpProjectZone()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Region: %s\nZone: %s\n", region, zone)

	fmt.Printf("STEP 2: lookup instance from IP address\n")
	instance, err := client.GcpInstanceName()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Instance: %s\n", instance)

	fmt.Printf("STEP 2a: prepare snapshot with GCS lambda store config\n")
	if err := createGcsTemplate(); err != nil {
		panic(fmt.Errorf("failed to create GCS template.json: %w", err))
	}

	fmt.Printf("STEP 3: take crash-consistent snapshot of instance\n")
	disk := instance // assume Gcp disk name is same as instance name
	resp, err := client.Wait(client.GcpSnapshot(disk, "boss-snap"))
	fmt.Println(resp)
	if err != nil {
		panic(err)
	}

	return &WorkerPool{
		WorkerPoolPlatform: &GcpWorkerPool{
			client: client,
		},
	}
}

func (_ *GcpWorkerPool) NewWorker(workerId string) *Worker {
	return &Worker{
		workerId: workerId,
		host:     "",
		port:     "5000",
	}
}

func (pool *GcpWorkerPool) CreateInstance(worker *Worker) error {
	client := pool.client
	fmt.Printf("creating new VM from snapshot\n")

	resp, err := client.Wait(client.LaunchGcp("boss-snap", worker.workerId)) // TODO: load snapshot name from Config

	if err != nil && resp["error"].(map[string]any)["code"] != "409" { // continue if instance already exists error
		fmt.Printf("instance alreay exists!\n")
		client.startGcpInstance(worker.workerId)
	} else if err != nil {
		return err
	}

	lookup, err := client.GcpInstancetoIP()
	if err != nil {
		return err
	}

	worker.host = lookup[worker.workerId]

	// TODO: check if runCmd fails and fix the ol-min hardcoding
	worker.runCmd("./ol worker up -d --image ol-min") 

	return nil
}

func (pool *GcpWorkerPool) DeleteInstance(worker *Worker) error {
	slog.Info(fmt.Sprintf("deleting gcp worker: %s", worker.workerId))
	worker.runCmd("./ol worker down")                                // TODO: check if runCmd fails
	pool.client.Wait(pool.client.deleteGcpInstance(worker.workerId)) // wait until instance is completely deleted

	return nil // TODO: check for error and make sure it is returning the error. We dont want delete failing and eating the error.
}

func (_ *GcpWorkerPool) ForwardTask(w http.ResponseWriter, r *http.Request, worker *Worker) error {
	return forwardTaskHelper(w, r, worker.host, worker.port)
}

// createGcsTemplate creates template.json with GCS registry configuration
// This will be captured in the snapshot and used by workers
func createGcsTemplate() error {
	currPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current path: %w", err)
	}

	templatePath := filepath.Join(currPath, "template.json")

	log.Printf("Creating template.json with GCS registry at: %s", templatePath)

	// Get default worker config
	defaultTemplateConfig, err := common.GetDefaultWorkerConfig("")
	if err != nil {
		return fmt.Errorf("failed to load default template config: %w", err)
	}

	// Set the GCS registry URL
	defaultTemplateConfig.Registry = config.BossConf.GetLambdaStoreURL()
	log.Printf("Setting template.json registry to: %s", defaultTemplateConfig.Registry)

	// Clear worker-specific fields so they get patched later
	defaultTemplateConfig.Worker_dir = ""
	defaultTemplateConfig.Pkgs_dir = ""
	defaultTemplateConfig.SOCK_base_path = ""
	defaultTemplateConfig.Import_cache_tree = ""
	defaultTemplateConfig.Worker_url = "0.0.0.0"

	// Save template.json with GCS registry
	if err := common.SaveConfig(defaultTemplateConfig, templatePath); err != nil {
		return fmt.Errorf("failed to save template.json: %w", err)
	}

	log.Printf("template.json with GCS registry ready for snapshot")
	return nil
}

package boss

import (
	"os"
	"fmt"
	"time"
	"context"
	"strconv"

	"github.com/digitalocean/godo"
)

// Globals
const BOSS_IDX int = 0 // ASSUME: Boss is the first VM created
const SNAPSHOT_NAME string = "boss snap"
const CHILD_NAME string = "tony"

type DOWorkerPool struct {
	Client *godo.Client
	BossVM godo.Droplet
	BossKey godo.Key
	BossSnap godo.Snapshot
	ParentPool *WorkerPool
}

type DOWorker struct {
	pool *DOWorkerPool
}

// Creates new worker pool
func NewDOWorkerPool() (*WorkerPool) {
	
	// Check API Token
	token := os.Getenv("DIGITALOCEAN_TOKEN")
	if len(token) == 0 {
		fmt.Println("ERROR: Unable to find a DigitalOcean personal access token.\n Generate token and export as environment variable named 'DIGITALOCEAN_TOKEN'\n(src: https://docs.digitalocean.com/reference/api/create-personal-access-token/)")
		panic()
	}
	
	// Establishing auth information
	client := godo.NewFromToken(token)
	ctx := context.TODO()

	// REST call body
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}

	// Verify SSH key is setup
	// Make GET: SSH information
	keys, _, err := client.Keys.List(ctx, opt)
	if err != nil {
		fmt.Println("ERROR: An error was encountered while listing SSH information. Aborting...\n")
		panic(err)
	} if len(keys) == 0 {
		fmt.Println("ERROR: Unable to find SSH Key setup.\n Setup SSH keys by visiting https://docs.digitalocean.com/products/droplets/how-to/add-ssh-keys/")
		panic()
	}
	boss_key := keys[BOSS_IDX]

	// Make GET: Boss Info
	droplets, _, err := client.Droplets.List(ctx, opt)
	if err != nil {
		fmt.Println("ERROR: An error was encountered while listing droplets. Aborting...")
		panic(err)
	} if len(droplets) == 0 {
		fmt.Println("ERROR: Unable to find boss. Make sure there is at least ONE VM in active state on your cloud dashboard.\n")
		panic()
	}
	boss_drop := droplets[BOSS_IDX]

	// TODO: Check for snapshot

	// Fill out DO Worker Pool
	DOpool := &DOWorkerPool {
		Client: client,
		BossVM: boss_drop,
		BossKey: boss_key,
	}
	parent := &WorkerPool {
		WorkerPoolPlatform: DOpool,
	}
	DOpool.ParentPool = parent
	return parent
}

// Defines new DO Worker
func (pool *DOWorkerPool) NewWorker(nextId int) *Worker {
	workerId := fmt.Sprintf("ol-worker-%d", nextId)
	return &Worker{
		workerId:       workerId,
		workerIp:       "",
		WorkerPlatform: DOWorker{},
		pool:           pool.ParentPool,
	}
}

// Creates new VM instance
func (pool *DOWorkerPool) CreateInstance(worker *Worker) {
	
	// Authenticate
	client := pool.Client
	ctx := context.TODO()

	fmt.Printf("Creating Droplet from: %v\n", SNAPSHOT_NAME)
	// Make POST: create Droplet
	create_request := &godo.DropletCreateRequest{
		Name:   CHILD_NAME,
		Region: BossSnap.Regions[BOSS_IDX],
		Size:   BossVM.Size.Slug,
		Image: godo.DropletCreateImage{
			ID: strconv.Atoi(BossSnap.ID),
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			{ID: BossKey.ID},
			{Fingerprint: BossKey.Fingerprint},
		},
	}
	/////////////////////// START
	// t0 := time.Now()
	child_drop, _, err:=client.Droplets.Create(ctx, create_request)
	if err != nil {
		fmt.Println("ERROR: An error was encountered while creating new Droplet. Aborting...\n")
		panic(err)
	} 
	fmt.Printf("Created Droplet %v. Waiting for request to complete...\n", child_drop.Name)
	status := child_drop.Status // status: 'new' after creation
	// Polling
	for status != "active" {
		// Sleep
		time.Sleep(1 * time.Second)

		// Make GET: Droplet information
		child_drop, _, err := client.Droplets.Get(ctx, child_drop.ID)
		if err != nil {
			fmt.Println("ERROR: An error was encountered while retrieving Droplet information. Aborting...\n")
			panic(err)
		}
		status = child_drop.Status // Keep looping
	}
	// CREATE_DROP = time.Since(t0)
	/////////////////////// END
	fmt.Printf("Wait complete. %v was created successfully\n", child_drop.Name)

	// Set workerID
	worker.workerIp = child_drop.ID
}


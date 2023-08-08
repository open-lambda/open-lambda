package cloudvm

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"
	"net/http"

	"github.com/digitalocean/godo"
)

// Globals
const BOSS_IDX int = 0 // ASSUME: Boss is the first VM created
const SNAPSHOT_NAME string = "boss snap"

type DOWorkerPool struct {
	Client     *godo.Client
	BossVM     godo.Droplet
	BossKey    godo.Key
	BossSnap   godo.Snapshot
	ParentPool *WorkerPool
}

// Helper function to click snapshots
func click_snap(client *godo.Client, droplet_id int, snap_name string) ([]godo.Snapshot, error) {

	// Etablishing auth info
	ctx := context.TODO()

	// Make POST: click Snapshot
	/////////////////////// START
	t0 := time.Now()
	action, _, err := client.DropletActions.Snapshot(ctx, droplet_id, snap_name)
	snap_act_id := action.ID
	status := action.Status
	fmt.Println("Clicked Snapshot. Waiting for request to complete...")

	// Polling
	for status != "completed" {

		// Sleep
		time.Sleep(1 * time.Second)

		// Make GET: Snapshot information
		action, _, err := client.DropletActions.Get(ctx, droplet_id, snap_act_id)
		if err != nil {
			return nil, err
		}
		status = action.Status // Keep looping
	}
	SNAPSHOT_DROP := time.Since(t0)
	/////////////////////// STOP
	fmt.Printf("Wait complete. Returning... Droplet Size: 4-80GB\nSnapshot_time: %d", SNAPSHOT_DROP)

	// Make GET: Snapshot Info
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 200,
	}
	snapshots, _, err := client.Snapshots.List(ctx, opt)

	// Return most recent snapshot slice
	return snapshots, err
}

// Creates new worker pool
func NewDOWorkerPool() *WorkerPool {

	// Check API Token
	token := os.Getenv("DIGITALOCEAN_TOKEN")
	if len(token) == 0 {
		err_msg := "ERROR: Unable to find a DigitalOcean personal access token.\n Generate token and export as environment variable named 'DIGITALOCEAN_TOKEN'\n(src: https://docs.digitalocean.com/reference/api/create-personal-access-token/)"
		panic(err_msg)
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
		err_msg := "ERROR: An error was encountered while listing SSH information. Aborting...\n"
		panic(err_msg)
	} else if len(keys) == 0 {
		err_msg := "ERROR: Unable to find SSH Key setup.\n Setup SSH keys by visiting https://docs.digitalocean.com/products/droplets/how-to/add-ssh-keys/"
		panic(err_msg)
	}
	boss_key := keys[BOSS_IDX]

	// Make GET: Boss Info
	droplets, _, err := client.Droplets.List(ctx, opt)
	if err != nil {
		fmt.Println("ERROR: An error was encountered while listing droplets. Aborting...")
		panic(err)
	} else if len(droplets) == 0 {
		err_msg := "ERROR: Unable to find boss. Make sure there is at least ONE VM in active state on your cloud dashboard.\n"
		panic(err_msg)
	}
	boss_drop := droplets[BOSS_IDX]

	// Make GET: Snapshot Info
	snapshots, _, err := client.Snapshots.List(ctx, opt)
	if err != nil {
		fmt.Println("ERROR: An error was encountered while listing Snapshot information. Aborting...\n")
		panic(err)
	} else if len(snapshots) == 0 {
		// If snapshot DNE, click new snapshot: ETA 6.5 min
		fmt.Println("No snapshots found! Click new snapshot of boss")
		snapshots, err = click_snap(client, boss_drop.ID, SNAPSHOT_NAME)
		if err != nil {
			fmt.Println("ERROR: An error was encountered while clicking a snapshot. Aborting...\nError Message:", err)
			panic(err)
		}
		fmt.Printf("New snapshot '%v' successfully clicked.\n", SNAPSHOT_NAME)
	} // Otherwise, use existing snapshot
	boss_snap := snapshots[BOSS_IDX]

	// Fill out DO Worker Pool
	DOpool := &DOWorkerPool{
		Client:   client,
		BossVM:   boss_drop,
		BossKey:  boss_key,
		BossSnap: boss_snap,
	}
	parent := &WorkerPool{
		WorkerPoolPlatform: DOpool,
	}
	DOpool.ParentPool = parent
	return parent
}

// Defines new DO Worker
func (pool *DOWorkerPool) NewWorker(workerId string) *Worker {
	return &Worker{
		workerId:       workerId,
		workerIp:       "",
		pool:           pool.ParentPool,
	}
}

// Creates new VM instance
func (pool *DOWorkerPool) CreateInstance(worker *Worker) {

	// Authenticate
	client := pool.Client
	ctx := context.TODO()

	fmt.Printf("Creating Droplet from: %v\n", SNAPSHOT_NAME)
	snap_id, err := strconv.Atoi(pool.BossSnap.ID)
	if err != nil {
		panic(err)
	}
	
	// Make POST: create Droplet
	create_request := &godo.DropletCreateRequest{
		Name:   worker.workerId,
		Region: pool.BossSnap.Regions[BOSS_IDX],
		Size:   pool.BossVM.Size.Slug,
		Image: godo.DropletCreateImage{
			ID: snap_id,
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			{ID: pool.BossKey.ID},
			{Fingerprint: pool.BossKey.Fingerprint},
		},
	}
	/////////////////////// START
	t0 := time.Now()
	child_drop, _, err := client.Droplets.Create(ctx, create_request)
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
		child_drop, _, err = client.Droplets.Get(ctx, child_drop.ID)
		if err != nil {
			fmt.Println("ERROR: An error was encountered while retrieving Droplet information. Aborting...\n")
			panic(err)
		}
		status = child_drop.Status // Keep looping
	}
	create := time.Since(t0)
	/////////////////////// END
	// // Uncomment for debugging
	// // Give more time for network
	// count := 100
	// for count > 0 {
	// 	if len(child_drop.Networks.V4) > 0 {break}
	// 	fmt.Println("Size: ", len(child_drop.Networks.V4))
	// 	time.Sleep(1 * time.Second)
	// 	count = count -1
	// }
	// fmt.Println("Networks: ", child_drop.Networks, "Droplet: ", child_drop)
	fmt.Printf("Wait complete. %v was created successfully\n", child_drop.Name)
	// Accessing private Ip
	pvt_ip, err := child_drop.PrivateIPv4()
	if err != nil {
		panic(err)
	}
	// Set workerID
	worker.workerIp = pvt_ip
	fmt.Printf("ip set successfully. ip: %v\n", pvt_ip)
	fmt.Printf("\nPerf Stats\nDroplet Size: 4GB,80GB\nCreation Time: %v", create)
}

// Destroys instance from DO Dashboard
func (pool *DOWorkerPool) DeleteInstance(worker *Worker) {
	// Authenticate
	client := pool.Client
	ctx := context.TODO()

	fmt.Printf("Deleting DO worker: %v\n", worker.workerId)

	// Wait until deletion completes
	// Make POST: destroy Droplet -- based on input flag
	// type casting
	worker_int, _ := strconv.Atoi(worker.workerId)
	_, err := client.Droplets.Delete(ctx, worker_int)
	if err != nil {
		fmt.Printf("ERROR: An error was encountered while destroying %v Droplet. Aborting...\n", worker.workerId)
		panic(err)
	}

	fmt.Printf("Deleted DO worker %v\n", worker_int)
}

func (pool *DOWorkerPool) ForwardTask(w http.ResponseWriter, r *http.Request, worker *Worker) {
	forwardTaskHelper(w, r, worker.workerIp)
}
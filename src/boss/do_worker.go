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
}

type DOWorker struct {
	//TODO: Fill out
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

	// Fill out DO Worker Pool
	return &WorkerPool {
		WorkerPoolPlatform: &DOWorkerPool {
			Client: client,
			BossVM: boss_drop,
			BossKey: boss_key,
		}
	}
}
}

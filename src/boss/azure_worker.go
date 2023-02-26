package boss

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"time"
)

type AzureWorkerPool struct {
	workerNum int
	workers   *map[string]*AzureWorker
	nextId    int
}

type AzureWorker struct {
	pool        *AzureWorkerPool
	workerId    string
	privateAddr string
	publicAddr  string
	reqChan     chan *Invocation
	exitChan    chan bool
}

func (pool *AzureWorkerPool) CreateWorker(reqChan chan *Invocation) {
	log.Printf("creating an azure worker\n")
	conf := AzureCreateVM()
	var private string
	var public string

	vmNum := conf.Resource_groups.Rgroup[0].Numvm
	private = *conf.Resource_groups.Rgroup[0].Net_ifc[vmNum-1].Properties.IPConfigurations[0].Properties.PrivateIPAddress
	publicWrap := conf.Resource_groups.Rgroup[0].Net_ifc[vmNum-1].Properties.IPConfigurations[0].Properties.PublicIPAddress
	if publicWrap == nil {
		public = ""
	} else {
		public = *publicWrap.Properties.IPAddress
	}

	worker := new(AzureWorker)
	worker.pool = pool
	worker.workerId = fmt.Sprintf("ol-worker-%d", pool.nextId)
	worker.privateAddr = private
	worker.publicAddr = public // If newly created one, this is ""
	worker.reqChan = reqChan
	worker.exitChan = make(chan bool)

	go worker.launch(private)
	go worker.task()

	pool.workerNum += 1
	pool.nextId += 1
	(*pool.workers)[worker.workerId] = worker
}

func NewAzureWorkerPool() (*AzureWorkerPool, error) {
	conf, err := ReadAzureConfig()
	if err != nil {
		log.Fatalln(err)
	}
	num := conf.Resource_groups.Rgroup[0].Numvm
	workers := make(map[string]*AzureWorker, num)
	pool := new(AzureWorkerPool)
	pool.workerNum = num
	pool.workers = &workers
	pool.nextId = num + 1
	for i := 0; i < num; i++ {
		worker_i := new(AzureWorker)
		worker_i.pool = pool
		worker_i.privateAddr = *conf.Resource_groups.Rgroup[0].Net_ifc[i].Properties.IPConfigurations[0].Properties.PrivateIPAddress
		publicWrap := conf.Resource_groups.Rgroup[0].Net_ifc[i].Properties.IPConfigurations[0].Properties.PublicIPAddress
		if publicWrap == nil {
			worker_i.publicAddr = ""
		} else {
			worker_i.publicAddr = *publicWrap.Properties.IPAddress
		}
		worker_i.workerId = *conf.Resource_groups.Rgroup[0].Vms[i].Name
	}
	return pool, nil
}

func (worker *AzureWorker) launch(privateIP string) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	cmd := fmt.Sprintf("cd %s; %s", cwd, "./ol worker --detach")
	tries := 10
	for tries > 0 {
		sshcmd := exec.Command("ssh", "-i", "~/.ssh/ol-boss_key.pem", user.Username+"@"+privateIP, "-o", "StrictHostKeyChecking=no", "-C", cmd)
		stdoutStderr, err := sshcmd.CombinedOutput()
		fmt.Printf("%s\n", stdoutStderr)
		if err == nil {
			break
		}
		tries -= 1
		if tries == 0 {
			fmt.Println(sshcmd.String())
			panic(err)
		}
		time.Sleep(5 * time.Second)
	}
}

func (worker *AzureWorker) task() {
	for {
		var req *Invocation
		select {
		case <-worker.exitChan:
			return
		case req = <-worker.reqChan:
		}

		if req == nil {
			worker.Close()
			return
		}

		err = forwardTask(req.w, req.r, worker.privateAddr)
		if err != nil {
			panic(err)
		}

		req.Done <- true
	}
}

func (worker *AzureWorker) Close() {
	worker.exitChan <- true
}

func (pool *AzureWorkerPool) CloseAll() {}

func (pool *AzureWorkerPool) DeleteWorker(worderId string) {}

func (pool *AzureWorkerPool) Size() int {
	return 0
}

func (pool *AzureWorkerPool) Status() []string {
	b := []string{"abc", "def"}
	return b
}

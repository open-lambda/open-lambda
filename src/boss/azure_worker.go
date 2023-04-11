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
	workerNum  int
	workers    *map[string]*AzureWorker
	nextId     int
	parentPool *WorkerPool
}

type AzureWorker struct {
	pool         *AzureWorkerPool
	parentWorker *Worker
	workerId     string
	configPosit  int
	diskName     string
	vnetName     string
	subnetName   string
	nsgName      string
	nicName      string
	publicIPName string
	privateAddr  string
	publicAddr   string
}

func NewAzureWorkerPool() (*WorkerPool, error) {
	conf, err := ReadAzureConfig()
	if err != nil {
		log.Fatalln(err)
	}
	num := conf.Resource_groups.Rgroup[0].Numvm
	workers := make(map[string]*AzureWorker, num)
	pool := &AzureWorkerPool{
		workerNum: num,
		workers:   &workers,
		nextId:    num + 1,
	}
	for i := 0; i < num; i++ {
		worker_i := new(AzureWorker)
		worker_i.pool = pool
		worker_i.privateAddr = *conf.Resource_groups.Rgroup[0].Vms[i].Net_ifc.Properties.IPConfigurations[0].Properties.PrivateIPAddress
		publicWrap := conf.Resource_groups.Rgroup[0].Vms[i].Net_ifc.Properties.IPConfigurations[0].Properties.PublicIPAddress
		if publicWrap == nil {
			worker_i.publicAddr = ""
		} else {
			worker_i.publicAddr = *publicWrap.Properties.IPAddress
		}
		worker_i.workerId = *conf.Resource_groups.Rgroup[0].Vms[i].Vm.Name
		worker_i.configPosit = num
	}
	parent := &WorkerPool{
		nextId:             1,
		workers:            map[string]*Worker{},
		queue:              make(chan *Worker, Conf.Worker_Cap),
		WorkerPoolPlatform: pool,
		startingWorkers:    make(map[string]*Worker),
		runningWorkers:     make(map[string]*Worker),
		cleaningWorkers:    make(map[string]*Worker),
		destroyingWorkers:  make(map[string]*Worker),
		needRestart:        false,
	}
	pool.parentPool = parent
	return parent, nil
}

// Is nextId here useful? I store nextId in the pool
// TODO: maybe store nextId to the config file so that if the boss shut down, it know how to do next time
func (pool *AzureWorkerPool) NewWorker(nextId int) *Worker {
	workerId := fmt.Sprintf("ol-worker-%d", nextId)
	return &Worker{
		workerId:       workerId,
		workerIp:       "",
		WorkerPlatform: AzureWorker{},
		pool:           pool.parentPool,
	}
}

// TODO: make AzureCreateVM multiple-threaded
func (pool *AzureWorkerPool) CreateInstance(worker *Worker) {
	log.Printf("creating an azure worker\n")
	conf := AzureCreateVM(worker)
	var private string

	pool.parentPool.lock.Lock()
	worker.numTask = 1
	vmNum := conf.Resource_groups.Rgroup[0].Numvm
	private = worker.workerIp
	newDiskName := worker.workerId + "-disk"
	newNicName := worker.workerId + "-nic"
	newNsgName := worker.workerId + "-nsg"
	subnetName := worker.workerId + "-subnet"
	vnetName := "ol-boss-vnet"
	publicIPName := ""
	public := ""

	azworker := &AzureWorker{
		pool:         pool,
		parentWorker: worker,
		workerId:     worker.workerId,
		configPosit:  vmNum - 1,
		diskName:     newDiskName,
		vnetName:     vnetName,
		nicName:      newNicName,
		nsgName:      newNsgName,
		subnetName:   subnetName,
		publicIPName: publicIPName,
		privateAddr:  private,
		publicAddr:   public, // If newly created one, this is ""
	}

	pool.workerNum += 1
	pool.nextId = pool.workerNum + 1

	(*pool.workers)[azworker.workerId] = azworker
	worker.workerId = azworker.workerId
	worker.workerIp = azworker.publicAddr
	worker.WorkerPlatform = azworker
	pool.parentPool.lock.Unlock()

	// start worker
	azworker.startWorker()

	pool.parentPool.lock.Lock()
	delete(pool.parentPool.startingWorkers, worker.workerId)
	pool.parentPool.runningWorkers[worker.workerId] = worker
	pool.parentPool.lock.Unlock()
}

func (worker *AzureWorker) startWorker() {
	worker.parentWorker.numTask = 1
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	cmd := fmt.Sprintf("cd %s; %s; %s", cwd, "sudo mount -o rw,remount /sys/fs/cgroup", "sudo ./ol worker --detach")
	tries := 10
	for tries > 0 {
		sshcmd := exec.Command("ssh", "-i", "~/.ssh/ol-boss_key.pem", user.Username+"@"+worker.privateAddr, "-o", "StrictHostKeyChecking=no", "-C", cmd)
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
	worker.parentWorker.numTask = 0
}

func (worker *AzureWorker) killWorker() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	cmd := fmt.Sprintf("cd %s; %s", cwd, "sudo ./ol kill")
	log.Printf("Try to ssh into the worker and kill the process")
	tries := 10
	for tries > 0 {
		log.Printf("debug: %s\n", worker.privateAddr)
		sshcmd := exec.Command("ssh", "-i", "~/.ssh/ol-boss_key.pem", user.Username+"@"+worker.privateAddr, "-o", "StrictHostKeyChecking=no", "-C", cmd)
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

func (pool *AzureWorkerPool) DeleteInstance(generalworker *Worker) {
	worker := (*pool.workers)[generalworker.workerId]
	log.Printf("Killing worker: %s", worker.workerId)

	worker.killWorker()

	pool.parentPool.lock.Lock()
	delete(pool.parentPool.cleaningWorkers, generalworker.workerId)
	pool.parentPool.cleanedWorker = generalworker
	pool.parentPool.updateCluster()
	if pool.parentPool.needRestart {
		pool.parentPool.lock.Unlock()
		worker.startWorker()
		return
	}
	pool.parentPool.lock.Unlock()

	// delete the vm
	log.Printf("Try to delete the vm")
	cleanupVM(worker)

	pool.parentPool.lock.Lock()
	// shrink length
	conf, _ := ReadAzureConfig()
	conf.Resource_groups.Rgroup[0].Numvm -= 1
	// shrink slice
	conf.Resource_groups.Rgroup[0].Vms[worker.configPosit] = conf.Resource_groups.Rgroup[0].Vms[len(conf.Resource_groups.Rgroup[0].Vms)-1]
	conf.Resource_groups.Rgroup[0].Vms = conf.Resource_groups.Rgroup[0].Vms[:conf.Resource_groups.Rgroup[0].Numvm]
	//fmt.Println(*conf.Resource_groups.Rgroup[0].Vms[worker.configPosit].Vm.Name)
	if len(conf.Resource_groups.Rgroup[0].Vms) > 0 && worker.configPosit < conf.Resource_groups.Rgroup[0].Numvm {
		// if all workers has been deleted, don't do this
		// if the worker to be deleted is at the end of the list, don't do this
		(*worker.pool.workers)[*conf.Resource_groups.Rgroup[0].Vms[worker.configPosit].Vm.Name].configPosit = worker.configPosit
	}
	worker.pool.workerNum -= 1
	WriteAzureConfig(conf)
	log.Printf("Deleted the worker and worker VM successfully\n")
	delete(pool.parentPool.destroyingWorkers, generalworker.workerId) // delete from the map
	// call updateCluster here
	pool.parentPool.destroyedWorker = generalworker
	pool.parentPool.updateCluster()
	pool.parentPool.lock.Unlock()
}

func (pool *AzureWorkerPool) Size() int {
	return len(pool.parentPool.startingWorkers) + len(pool.parentPool.runningWorkers)
}

func (pool *AzureWorkerPool) Status() []string {
	var w = []string{}
	for k, _ := range *pool.workers {
		w = append(w, k)
	}
	return w
}

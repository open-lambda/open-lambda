package boss

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"sync"
	"time"
)

type AzureWorkerPool struct {
	workerNum int
	workers   *map[string]*AzureWorker
	nextId    int
}

type AzureWorker struct {
	pool         *AzureWorkerPool
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

func NewAzureWorkerPool() *WorkerPool {
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
		WorkerPoolPlatform: pool,
	}
	return parent
}

// Is nextId here useful? I store nextId in the pool
// TODO: maybe store nextId to the config file so that if the boss shut down, it know how to do next time
func (pool *AzureWorkerPool) NewWorker(workerId string) *Worker {
	return &Worker{
		workerId: workerId,
		workerIp: "",
	}
}

// TODO: make AzureCreateVM multiple-threaded
func (pool *AzureWorkerPool) CreateInstance(worker *Worker) error {
	log.Printf("creating an azure worker\n")
	conf, err := AzureCreateVM(worker)
	if err != nil {
		return err
	}
	var private string

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
	worker.workerIp = azworker.privateAddr

	return nil
}

func (worker *Worker) startWorker() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	// cmd1 := fmt.Sprintf("cd %s; %s; %s",
	// 	cwd,
	// 	"sudo mount -o rw,remount /sys/fs/cgroup",
	// 	"sudo ./ol worker init -i ol-min")
	// cmd2 := fmt.Sprintf("cd %s; %s; %s", cwd,
	// 	"sudo ./ol worker up -i ol-min -d",
	// 	"sudo ./ol bench init")
	cmd := fmt.Sprintf("cd %s; %s; %s",
		cwd,
		"sudo mount -o rw,remount /sys/fs/cgroup",
		"sudo ./ol worker up -i ol-min -d")
	tries := 10
	for tries > 0 {
		sshcmd := exec.Command("ssh", "-i", "~/.ssh/ol-boss_key.pem", user.Username+"@"+worker.workerIp, "-o", "StrictHostKeyChecking=no", "-C", cmd)
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

	// for tries > 0 {
	// 	sshcmd := exec.Command("ssh", "-i", "~/.ssh/ol-boss_key.pem", user.Username+"@"+worker.workerIp, "-o", "StrictHostKeyChecking=no", "-C", cmd2)
	// 	stdoutStderr, err := sshcmd.CombinedOutput()
	// 	fmt.Printf("%s\n", stdoutStderr)
	// 	if err == nil {
	// 		break
	// 	}
	// 	tries -= 1
	// 	if tries == 0 {
	// 		fmt.Println(sshcmd.String())
	// 		panic(err)
	// 	}
	// 	time.Sleep(5 * time.Second)
	// }

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
	cmd := fmt.Sprintf("cd %s; %s", cwd, "sudo ./ol worker down")
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

var conf_lock sync.Mutex

func (pool *AzureWorkerPool) DeleteInstance(generalworker *Worker) error {
	worker := (*pool.workers)[generalworker.workerId]

	// delete the vm
	log.Printf("Try to delete the vm")
	worker.killWorker()
	cleanupVM(worker)

	// shrink length
	conf_lock.Lock()
	conf, _ := ReadAzureConfig()
	conf.Resource_groups.Rgroup[0].Numvm -= 1
	// shrink slice
	conf.Resource_groups.Rgroup[0].Vms[worker.configPosit] = conf.Resource_groups.Rgroup[0].Vms[len(conf.Resource_groups.Rgroup[0].Vms)-1]
	conf.Resource_groups.Rgroup[0].Vms = conf.Resource_groups.Rgroup[0].Vms[:conf.Resource_groups.Rgroup[0].Numvm]
	//fmt.Println(*conf.Resource_groups.Rgroup[0].Vms[worker.configPosit].Vm.Name)
	if len(conf.Resource_groups.Rgroup[0].Vms) > 0 && worker.configPosit < conf.Resource_groups.Rgroup[0].Numvm {
		// if all workers has been deleted, don't do this
		// if the worker to be deleted is at the end of the list, don't do this

		//TODO: fix this..?
		(*worker.pool.workers)[*conf.Resource_groups.Rgroup[0].Vms[worker.configPosit].Vm.Name].configPosit = worker.configPosit
	}
	worker.pool.workerNum -= 1
	WriteAzureConfig(conf)
	log.Printf("Deleted the worker and worker VM successfully\n")
	conf_lock.Unlock()

	return nil
}

func (pool *AzureWorkerPool) ForwardTask(w http.ResponseWriter, r *http.Request, worker *Worker) {
	forwardTaskHelper(w, r, worker.workerIp)
}

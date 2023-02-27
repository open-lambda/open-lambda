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
	pool         *AzureWorkerPool
	workerId     string
	vmName       string
	diskName     string
	vnetName     string
	subnetName   string
	nsgName      string
	nicName      string
	publicIPName string
	privateAddr  string
	publicAddr   string
	reqChan      chan *Invocation
	exitChan     chan bool
}

func (pool *AzureWorkerPool) CreateWorker(reqChan chan *Invocation) {
	log.Printf("creating an azure worker\n")
	conf := AzureCreateVM()
	var private string
	var public string

	vmNum := conf.Resource_groups.Rgroup[0].Numvm
	private = *conf.Resource_groups.Rgroup[0].Vms[vmNum-1].Net_ifc.Properties.IPConfigurations[0].Properties.PrivateIPAddress
	publicWrap := conf.Resource_groups.Rgroup[0].Vms[vmNum-1.].Net_ifc.Properties.IPConfigurations[0].Properties.PublicIPAddress
	if publicWrap == nil {
		public = ""
	} else {
		public = *publicWrap.Properties.IPAddress
	}

	worker := &AzureWorker{
		pool:         pool,
		workerId:     fmt.Sprintf("ol-worker-%d", pool.nextId),
		vmName:       vmName,
		diskName:     diskName,
		vnetName:     vnetName,
		nicName:      nicName,
		nsgName:      nsgName,
		subnetName:   subnetName,
		publicIPName: publicIPName,
		privateAddr:  private,
		publicAddr:   public, // If newly created one, this is ""
		reqChan:      reqChan,
		exitChan:     make(chan bool),
	}

	pool.workerNum += 1
	pool.nextId += 1

	go worker.launch()
	go worker.task()

	(*pool.workers)[worker.workerId] = worker

}

func NewAzureWorkerPool() (*AzureWorkerPool, error) {
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
	}
	return pool, nil
}

func (worker *AzureWorker) launch() {
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
			log.Printf("Prepare to close the worker\n")
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
	select {
	case worker.exitChan <- true:
	default:
		log.Printf("%t", <-worker.exitChan)
	}
	log.Printf("Closing worker: %s; vm: %s\n", worker.workerId, worker.vmName)
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
	// delete the vm
	log.Printf("Try to delete the vm")
	cleanupVM(worker)
	// remove the deleted vm information from the config file
	log.Printf("Try to remove the vm information from the config file")
	conf, _ := ReadAzureConfig()
	for i, value := range conf.Resource_groups.Rgroup[0].Vms {
		vm := value.Vm.Name
		if *vm == worker.vmName {
			// delete this one
			for j := i; j < len(conf.Resource_groups.Rgroup[0].Vms)-1; j++ {
				conf.Resource_groups.Rgroup[0].Vms[j] = conf.Resource_groups.Rgroup[0].Vms[j+1]
			}
			break
		}
	}
	conf.Resource_groups.Rgroup[0].Numvm -= 1
	WriteAzureConfig(conf)
	log.Printf("Deleted the worker and worker VM successfully\n")
}

func (pool *AzureWorkerPool) CloseAll() {
	for _, w := range *pool.workers {
		w.Close()
	}
}

func (pool *AzureWorkerPool) DeleteWorker(worderId string) {}

func (pool *AzureWorkerPool) Size() int {
	conf, _ := ReadAzureConfig()
	return conf.Resource_groups.Rgroup[0].Numvm
}

func (pool *AzureWorkerPool) Status() []string {
	var w = []string{}
	for k, _ := range *pool.workers {
		w = append(w, k)
	}
	return w
}

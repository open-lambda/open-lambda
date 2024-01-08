package cloudvm

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
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

func NewAzureWorkerPool() (*WorkerPool, error) {
	conf, err := ReadAzureConfig()
	if err != nil {
		return nil, err
	}
	if len(conf.Resource_groups.Rgroup) != 1 {
		err1 := errors.New("should have one resource group")
		return nil, err1
	}
	num := conf.Resource_groups.Rgroup[0].Numvm
	workers := make(map[string]*AzureWorker, num)
	pool := &AzureWorkerPool{
		workerNum: num,
		workers:   &workers,
		nextId:    num + 1,
	}
	for i := 0; i < num; i++ {
		cur_vm := conf.Resource_groups.Rgroup[0].Vms[i]
		worker_i := &AzureWorker{
			pool:        pool,
			privateAddr: *cur_vm.Net_ifc.Properties.IPConfigurations[0].Properties.PrivateIPAddress,
			workerId:    *cur_vm.Vm.Name,
			configPosit: num,
		}
		publicWrap := conf.Resource_groups.Rgroup[0].Vms[i].Net_ifc.Properties.IPConfigurations[0].Properties.PublicIPAddress
		if publicWrap == nil {
			worker_i.publicAddr = ""
		} else {
			worker_i.publicAddr = *publicWrap.Properties.IPAddress
		}
	}
	parent := &WorkerPool{
		WorkerPoolPlatform: pool,
	}
	return parent, nil
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

	vmNum := conf.Resource_groups.Rgroup[0].Numvm
	private := worker.workerIp
	newDiskName := worker.workerId + "-disk"
	newNicName := worker.workerId + "-nic"
	newNsgName := worker.workerId + "-nsg"
	subnetName := worker.workerId + "-subnet"
	vnetName := "ol-boss-new-vnet"
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

func (worker *Worker) start(firstTime bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// user, err := user.Current()
	if err != nil {
		panic(err)
	}

	pid_namespace := "echo 500000 | sudo tee /proc/sys/user/max_pid_namespaces"
	ipc_namespace := "echo 500000 | sudo tee /proc/sys/user/max_ipc_namespaces"
	uts_namespace := "echo 500000 | sudo tee /proc/sys/user/max_uts_namespaces"

	python_path := test_path

	workerNum := len(worker.pool.workers[RUNNING]) + len(worker.pool.workers[STARTING])
	run_deploy_funcs := fmt.Sprintf("sudo python3 write_funcs.py %d", workerNum)

	run_one_time := "sudo python3 run_worker.py"

	var run_worker_up string
	run_worker_up = fmt.Sprintf("sudo ./ol worker up -i ol-min -d -o import_cache_tree=%s,worker_url=0.0.0.0,features.warmup=false,limits.mem_mb=500,mem_pool_mb=8192,trace.evictor=true", tree_path)

	var cmd string
	if firstTime {
		cmd = fmt.Sprintf("%s; %s; %s; %s; cd %s; %s; cd %s; %s; %s; cd %s; %s",
			"sudo mount -o rw,remount /sys/fs/cgroup",
			pid_namespace,
			ipc_namespace,
			uts_namespace,
			cwd,
			"sudo ./ol worker init -i ol-min",
			python_path,
			run_one_time,
			run_deploy_funcs,
			cwd,
			run_worker_up,
		)
	} else {
		cmd = fmt.Sprintf("%s; cd %s; %s",
			"sudo mount -o rw,remount /sys/fs/cgroup",
			cwd,
			run_worker_up,
		)
	}

	tries := 5
	for tries > 0 {
		sshcmd := exec.Command("ssh", "-i", ssh_key_path, "azureuser"+"@"+worker.workerIp, "-o", "StrictHostKeyChecking=no", "-C", cmd)
		stdoutStderr, err := sshcmd.CombinedOutput()
		fmt.Printf("%s\n", stdoutStderr)
		if err == nil {
			break
		}
		tries -= 1
		if tries == 0 {
			log.Println("sshing into the worker:", sshcmd.String())
			return err
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

func (worker *Worker) killWorker() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// user, err := user.Current()
	if err != nil {
		panic(err)
	}
	cmd := fmt.Sprintf("cd %s; %s", cwd, "sudo ./ol worker down")
	log.Printf("Try to ssh into the worker and kill the process")
	tries := 10
	for tries > 0 {
		log.Printf("debug: %s\n", worker.workerIp)
		sshcmd := exec.Command("ssh", "-i", ssh_key_path, "azureuser"+"@"+worker.workerIp, "-o", "StrictHostKeyChecking=no", "-C", cmd)
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
	generalworker.killWorker()
	cleanupVM(worker)

	// shrink length
	conf_lock.Lock()
	defer conf_lock.Unlock()

	conf, _ := ReadAzureConfig()
	conf.Resource_groups.Rgroup[0].Numvm -= 1
	// shrink slice
	conf.Resource_groups.Rgroup[0].Vms[worker.configPosit] = conf.Resource_groups.Rgroup[0].Vms[len(conf.Resource_groups.Rgroup[0].Vms)-1]
	conf.Resource_groups.Rgroup[0].Vms = conf.Resource_groups.Rgroup[0].Vms[:conf.Resource_groups.Rgroup[0].Numvm]
	if len(conf.Resource_groups.Rgroup[0].Vms) > 0 && worker.configPosit < conf.Resource_groups.Rgroup[0].Numvm {
		// if all workers has been deleted, don't do this
		// if the worker to be deleted is at the end of the list, don't do this

		//TODO: fix this..?
		(*worker.pool.workers)[*conf.Resource_groups.Rgroup[0].Vms[worker.configPosit].Vm.Name].configPosit = worker.configPosit
	}
	worker.pool.workerNum -= 1
	WriteAzureConfig(conf)
	log.Printf("Deleted the worker and worker VM successfully\n")

	return nil
}

func (pool *AzureWorkerPool) ForwardTask(w http.ResponseWriter, r *http.Request, worker *Worker) {
	err := forwardTaskHelper(w, r, worker.workerIp)
	if err != nil {
		log.Printf("%s", err.Error())
	}
}

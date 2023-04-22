func (pool *GcpWorkerPool) CreateInstance(worker *Worker) {
        // Authenticate

        fmt.Printf("Creating Droplet from boss snapshot\n")

        // Check if instance already exists

        // request body
	
	// Make POST: create droplet

	// Set workerIp
        worker.workerIp = CHILD DROPLET Ip
}

//this function will only destroy instance from cloud platform
func (pool *GcpWorkerPool) DeleteInstance(worker *Worker) {
        // Authenticate

        fmt.Printf("Deleting DO worker: %v\n", worker.workerId)

        // Wait until deletion completes

        fmt.Printf("Deleted DO worker %v\n", worker.worker.Id)
}

package boss

import (
	"log"
	"time"
	"sync"
)

const (
	UPPERBOUND = 80
	LOWERBOUND = 30
	INACTIVITY_TIMEOUT = 60
)

type Scaling interface {
	Scale(pool *WorkerPool)
}

type ScalingThreshold struct {
	timeout		*time.Timer
	sync.Mutex
}

func (s *ScalingThreshold) Scale(pool *WorkerPool) {
	s.Lock()
	defer s.Unlock()
	
	if s.timeout != nil {
		s.timeout.Stop()
	}

	sumTask := 0
	numWorker := len(pool.workers[RUNNING]) + len(pool.workers[STARTING])
	
	for _, worker := range pool.workers[RUNNING] {
		sumTask += int(worker.numTask)
	}

	tasksPerWorker := sumTask/numWorker

	if pool.target < Conf.Worker_Cap && tasksPerWorker > UPPERBOUND {
		new_target := pool.target + tasksPerWorker/UPPERBOUND
		log.Println("scale up (target=%d)\n", new_target)
		pool.SetTarget(new_target)
	}

	if pool.target > 2 && tasksPerWorker < LOWERBOUND {
		new_target := pool.target - tasksPerWorker/LOWERBOUND
		log.Println("scale down (target=%d)\n", new_target)
		pool.SetTarget(new_target)
	}

	s.timeout = time.AfterFunc(INACTIVITY_TIMEOUT*time.Second, func() {
		log.Printf("scale down due to inactivity\n")
		pool.SetTarget(1)
	})
}
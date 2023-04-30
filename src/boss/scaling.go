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
	
	for _, worker := range pool.workers[RUNNING] {
		sumTask += int(worker.numTask)
	}

	tasksPerWorker := sumTask/pool.target

	if pool.target < Conf.Worker_Cap && tasksPerWorker > UPPERBOUND {
		new_target := (tasksPerWorker * pool.target)/UPPERBOUND + 1
		if new_target > Conf.Worker_Cap {
			new_target = Conf.Worker_Cap
		}
		log.Printf("scale up (target=%d)\n", new_target)
		pool.SetTarget(new_target)
	}

	if pool.target > 1 && tasksPerWorker < LOWERBOUND {
		new_target := (tasksPerWorker * pool.target)/LOWERBOUND 
		if new_target < 1 {
			new_target = 1
		}

		log.Printf("scale down (target=%d)\n", new_target)
		pool.SetTarget(new_target)
	}

	s.timeout = time.AfterFunc(INACTIVITY_TIMEOUT*time.Second, func() {
		if pool.target > 1 {
			log.Printf("scale down due to inactivity\n")
			pool.SetTarget(1)
		}
	})
}
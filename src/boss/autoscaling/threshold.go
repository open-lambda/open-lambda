package autoscaling

import (
	"log"
	"time"
	"github.com/open-lambda/open-lambda/ol/boss/cloudvm"
)

const (
	UPPERBOUND         = 80
	LOWERBOUND         = 30
	INACTIVITY_TIMEOUT = 60
)

type ThresholdScaling struct {
	pool *cloudvm.WorkerPool
	timeout *time.Timer
	exitChan chan bool
}

func (s *ThresholdScaling) Launch(pool *cloudvm.WorkerPool) {
	s.pool = pool
	s.exitChan = make(chan bool)
	log.Println("lauching threshold-scaler")
	pool.SetTarget(1) // initial cluster size set to 1
	go func() {
		for {
			s.Scale()
			time.Sleep(5 * time.Second)

			select {
			case _ = <-s.exitChan:
				return
			default:
			}
		}
	}()
}

func (s *ThresholdScaling) Scale() {
	pool := s.pool
	tasksPerWorker := pool.StatusTasks()["task/worker"]

	if pool.GetTarget() < pool.GetCap() && tasksPerWorker > UPPERBOUND {
		new_target := pool.GetTarget() + tasksPerWorker/UPPERBOUND
		log.Println("scale up (target=%d)\n", new_target)
		pool.SetTarget(new_target)
	}

	if pool.GetTarget() > 1 && tasksPerWorker < LOWERBOUND {
		new_target := pool.GetTarget() - (LOWERBOUND / tasksPerWorker)
		if new_target < 1 {
			new_target = 1
		}

		log.Println("scale down (target=%d)\n", new_target)
		pool.SetTarget(new_target)
	}

	s.timeout = time.AfterFunc(INACTIVITY_TIMEOUT*time.Second, func() {
		if pool.GetTarget() > 1 {
			log.Printf("scale down due to inactivity\n")
			pool.SetTarget(1)
		}
	})
}

func (s *ThresholdScaling) Close() {
	log.Println("stopping threshold-scaler")
	s.exitChan <- true
}

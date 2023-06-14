package boss

import (
	"log"
	"time"
)

const (
	UPPERBOUND         = 80
	LOWERBOUND         = 30
	INACTIVITY_TIMEOUT = 60
)

type ThresholdScaling struct {
	boss *Boss
	timeout *time.Timer
	exitChan chan bool
}

func (s *ThresholdScaling) Launch(b *Boss) {
	s.boss = b
	s.exitChan = make(chan bool)
	log.Println("lauching threshold-scaler")
	b.workerPool.SetTarget(1) //initial cluster size set to 1
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

	pool := s.boss.workerPool
	tasksPerWorker := s.boss.workerPool.StatusTasks()["task/worker"]

	if pool.target < Conf.Worker_Cap && tasksPerWorker > UPPERBOUND {
		new_target := pool.target + tasksPerWorker/UPPERBOUND
		log.Println("scale up (target=%d)\n", new_target)
		pool.SetTarget(new_target)
	}

	if pool.target > 1 && tasksPerWorker < LOWERBOUND {
		new_target := pool.target - (LOWERBOUND / tasksPerWorker)
		if new_target < 1 {
			new_target = 1
		}

		log.Println("scale down (target=%d)\n", new_target)
		pool.SetTarget(new_target)
	}

	s.timeout = time.AfterFunc(INACTIVITY_TIMEOUT*time.Second, func() {
		if pool.target > 1 {
			log.Printf("scale down due to inactivity\n")
			pool.SetTarget(1)
		}
	})
}

func (s *ThresholdScaling) Close() {
	log.Println("stopping threshold-scaler")
	s.exitChan <- true
}
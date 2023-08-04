package autoscaling

import (
	"log"
	"time"
	"github.com/open-lambda/open-lambda/ol/boss/cloudvm"
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
	pool.SetTarget(1) //initial cluster size set to 1
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
	// TODO
}

func (s *ThresholdScaling) Close() {
	// TODO
}

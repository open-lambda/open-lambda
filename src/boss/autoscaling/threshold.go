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
	// TODO
} 

func (s *ThresholdScaling) Scale() {
	// TODO
}

func (s *ThresholdScaling) Close() {
	// TODO
}

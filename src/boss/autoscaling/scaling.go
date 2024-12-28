// Package autoscaling provides the interface for different
// implementations of the scaling logic
package autoscaling

import "github.com/open-lambda/open-lambda/ol/boss/cloudvm"

type Scaling interface {
	Launch(pool *cloudvm.WorkerPool) // launch auto-scaler
	Scale() // makes scaling decision based on cluster status
	Close() // close auto-scaler
}

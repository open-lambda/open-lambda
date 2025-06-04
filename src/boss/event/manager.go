package event

import (
	"github.com/open-lambda/open-lambda/ol/boss/cloudvm"
	"github.com/open-lambda/open-lambda/ol/common"
)

type Manager struct {
	cronScheduler *CronScheduler
}

func NewManager(pool *cloudvm.WorkerPool) *Manager {
	return &Manager{
		cronScheduler: NewCronScheduler(pool),
	}
}

func (m *Manager) Register(functionName string, triggers common.Triggers) error {
	// clean up stale cron job
	m.Unregister(functionName)
	return m.cronScheduler.Register(functionName, triggers.Cron)
}

func (m *Manager) Unregister(functionName string) {
	m.cronScheduler.Unregister(functionName)
}

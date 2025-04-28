package event

import (
	"github.com/open-lambda/open-lambda/ol/common"
)

type Manager struct {
	cronScheduler *CronScheduler
}

func NewManager() *Manager {
	return &Manager{
		cronScheduler: NewCronScheduler(),
	}
}

func (m *Manager) Register(functionName string, triggers common.Triggers) {
	if len(triggers.Cron) > 0 {
		m.cronScheduler.Register(functionName, triggers.Cron)
	}
}

func (m *Manager) Unregister(functionName string) {
	m.cronScheduler.Unregister(functionName)
}

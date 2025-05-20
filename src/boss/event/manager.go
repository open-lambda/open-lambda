package event

import (
	"github.com/open-lambda/open-lambda/ol/common"
)

type Trigger interface {
	Register(functionName string)
	Unregister(functionName string)
}

type Manager struct {
	cronScheduler *CronScheduler
	kafkaManager  *KafkaManager
}

func NewManager() *Manager {
	return &Manager{
		cronScheduler: NewCronScheduler(),
		kafkaManager:  NewKafkaManager(),
	}
}

func (m *Manager) Register(functionName string, triggers common.Triggers) {
	if len(triggers.Cron) > 0 {
		m.cronScheduler.Register(functionName, triggers.Cron)
	}

	if len(triggers.Kafka) > 0 {
		m.kafkaManager.Register(functionName, triggers.Kafka)
	}
}

func (m *Manager) Unregister(functionName string) {
	m.cronScheduler.Unregister(functionName)
	m.kafkaManager.Unregister(functionName)
}

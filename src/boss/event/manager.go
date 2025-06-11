package event

import (
	"fmt"

	"github.com/open-lambda/open-lambda/ol/boss/cloudvm"
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

func NewManager(pool *cloudvm.WorkerPool) *Manager {
	return &Manager{
		cronScheduler: NewCronScheduler(pool),
		kafkaManager:  NewKafkaManager(pool),
	}
}

// Register installs new triggers for the given lambda.
//
// This method is not concurrency-safe per function and must be called
// while holding the corresponding LambdaEntry.Lock in LambdaStore.
//
// It automatically calls Unregister first to clean up any stale triggers.
func (m *Manager) Register(functionName string, triggers common.Triggers) error {
	// Clean up any stale triggers
	m.Unregister(functionName)

	// Register cron triggers
	if err := m.cronScheduler.Register(functionName, triggers.Cron); err != nil {
		return fmt.Errorf("failed to register cron triggers: %w", err)
	}

	// Register Kafka triggers
	if err := m.kafkaManager.Register(functionName, triggers.Kafka); err != nil {
		return fmt.Errorf("failed to register Kafka triggers: %w", err)
	}

	return nil
}

// Unregister removes all active triggers for the given lambda.
//
// This method must also be called while holding the LambdaEntry.Lock,
// to avoid race conditions with concurrent registration or deletion.
func (m *Manager) Unregister(functionName string) error {
	m.cronScheduler.Unregister(functionName)

	if err := m.kafkaManager.Unregister(functionName); err != nil {
		return fmt.Errorf("failed to unregister kafka trigger for %s: %w", functionName, err)
	}

	return nil
}

package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/open-lambda/open-lambda/go/boss/cloudvm"
	"github.com/open-lambda/open-lambda/go/common"
)

// NOTE: The worker-side implementation of Kafka triggers is still in progress.
// This logic may need to be revised once the worker-side behavior is finalized.

type KafkaManager struct {
	workerPool *cloudvm.WorkerPool // to forward the req to worker
	lock       sync.Mutex
	functions  map[string]KafkaFunctionEntry // Maps function names to their active Kafka triggers and the worker addresses where the consumers were initialized
}

type KafkaFunctionEntry struct {
	Triggers []common.KafkaTrigger
	Worker   *cloudvm.Worker
}

func NewKafkaManager(pool *cloudvm.WorkerPool) *KafkaManager {
	return &KafkaManager{
		workerPool: pool,
		functions:  make(map[string]KafkaFunctionEntry),
	}
}

// Installs Kafka triggers for a given function.
func (k *KafkaManager) Register(functionName string, triggers []common.KafkaTrigger) error {
	if len(triggers) == 0 {
		return nil
	}

	k.lock.Lock()
	defer k.lock.Unlock()

	// TODO: Should all Kafka consumers for a function be assigned to a single worker,
	// or should each trigger be placed on a separate (possibly different) worker?
	// TODO: Add smarter worker selection, load balancing
	selectedWorker, err := k.workerPool.GetWorker()
	if err != nil {
		return fmt.Errorf("failed to get the worker to setup kafka consumer: %w", err)
	}

	workerAddress, err := cloudvm.GetWorkerAddress(selectedWorker)
	if err != nil {
		return fmt.Errorf("[KafkaManager] Failed to get worker address: %w", err)
	}

	// Setup each trigger at the worker
	for _, trigger := range triggers {
		// validate the required fields, make sure it is not empty
		if len(trigger.Topics) == 0 || len(trigger.BootstrapServers) == 0 {
			return fmt.Errorf("invalid Kafka trigger for %s: must include at least one topic and one bootstrap server", functionName)
		}

		data, err := json.Marshal(trigger)
		if err != nil {
			return fmt.Errorf("failed to marshal Kafka trigger for %s: %w", functionName, err)
		}

		url := fmt.Sprintf("http://%s/kafka-init/%s", workerAddress, functionName)

		httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(data))
		if err != nil {
			return fmt.Errorf("failed to create request for %s: %w", functionName, err)
		}

		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			return fmt.Errorf("failed to send Kafka setup for %s: %w", functionName, err)
		}

		if resp.StatusCode != http.StatusOK {
			// TODO: try again on error?
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return fmt.Errorf("worker returned error on setup for %s: %s - %s", functionName, resp.Status, string(body))
		}

		resp.Body.Close()
	}

	// Track all triggers
	k.functions[functionName] = KafkaFunctionEntry{
		Triggers: triggers,
		Worker:   selectedWorker,
	}
	slog.Info(fmt.Sprintf("[KafkaManager] Kafka consumer(s) registered for %s", functionName))

	return nil
}

// Cleans up Kafka triggers previously registered for a given function.
func (k *KafkaManager) Unregister(functionName string) error {
	k.lock.Lock()
	defer k.lock.Unlock()

	entry, exists := k.functions[functionName]
	if !exists {
		return nil
	}

	workerAddress, err := cloudvm.GetWorkerAddress(entry.Worker)
	if err != nil {
		slog.Error(fmt.Sprintf("[KafkaManager] Failed to get worker address for %s: %v", functionName, err))
		return fmt.Errorf("get worker address failed: %w", err)
	}

	url := fmt.Sprintf("http://%s/kafka-stop/%s", workerAddress, functionName)

	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		slog.Error(fmt.Sprintf("[KafkaManager] Failed to create unsetup request for %s: %v", functionName, err))
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		slog.Error(fmt.Sprintf("[KafkaManager] Failed to send Kafka unsetup request for %s: %v", functionName, err))
		return fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO: try again on error up to 3 times maybe?
		body, _ := io.ReadAll(resp.Body)
		slog.Error(fmt.Sprintf("[KafkaManager] Worker returned error on unsetup for %s: %s - %s", functionName, resp.Status, string(body)))
		return fmt.Errorf("unsetup failed with status: %s", resp.Status)
	}

	delete(k.functions, functionName)
	slog.Info("[KafkaManager] Kafka consumer unregistered for %s", functionName)
	return nil
}

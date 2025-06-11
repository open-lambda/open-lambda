package event

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/open-lambda/open-lambda/ol/boss/cloudvm"
	"github.com/open-lambda/open-lambda/ol/common"
)

type KafkaManager struct {
	workerPool *cloudvm.WorkerPool // to forward the req to worker
	mapLock    sync.Mutex
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

func (k *KafkaManager) Register(functionName string, triggers []common.KafkaTrigger) error {
	if len(triggers) == 0 {
		return nil
	}

	k.mapLock.Lock()
	defer k.mapLock.Unlock()

	// select 1 worker for all the kafka consumers? or try pick different one for each consumer?
	selectedWorker, err := k.workerPool.GetWorker()

	if err != nil {
		return fmt.Errorf("failed to get the worker to setup kafka consumer: %w", err)
	}

	// Setup each trigger at the worker
	for _, trigger := range triggers {
		// validate the required fields, make sure it is not empty
		if len(trigger.Topics) == 0 || len(trigger.Bootstrap_servers) == 0 {
			log.Printf("[KafkaManager] Skipping empty Kafka trigger for %s", functionName)
			continue
		}

		data, err := json.Marshal(trigger)
		if err != nil {
			log.Printf("[KafkaManager] Failed to marshal Kafka trigger for %s: %v", functionName, err)
			continue
		}

		req := httptest.NewRequest(http.MethodPost, "/kafka-init/"+functionName, bytes.NewBuffer(data))
		w := httptest.NewRecorder()

		err = k.workerPool.ForwardTask(w, req, selectedWorker)
		if err != nil {
			log.Printf("[KafkaManager] Failed to forward Kafka setup for %s: %v", functionName, err)
			continue
		}

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("[KafkaManager] Worker returned error on setup for %s: %s - %s", functionName, resp.Status, string(body))
		}

		resp.Body.Close()
	}

	// Track all triggers
	k.functions[functionName] = KafkaFunctionEntry{
		Triggers: triggers,
		Worker:   selectedWorker,
	}
	log.Printf("[KafkaManager] Kafka consumer(s) registered for %s", functionName)

	return nil
}

func (k *KafkaManager) Unregister(functionName string) error {
	k.mapLock.Lock()
	defer k.mapLock.Unlock()

	entry, exists := k.functions[functionName]
	if !exists {
		return nil
	}

	// Construct simulated request
	req := httptest.NewRequest(http.MethodPost, "/kafka-stop/"+functionName, bytes.NewBuffer([]byte(`{}`)))
	w := httptest.NewRecorder()

	// Assumes the worker is still alive during unregister; what happens if the worker is shutdown between register and unregister?
	err := k.workerPool.ForwardTask(w, req, entry.Worker)
	if err != nil {
		log.Printf("[KafkaManager] Failed to forward unsetup request for %s: %v", functionName, err)
		return err
	}

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// TODO: try again on error up to 3 times maybe?
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[KafkaManager] Worker returned error on unsetup for %s: %s - %s", functionName, resp.Status, string(body))
		return fmt.Errorf("unsetup failed with status: %s", resp.Status)
	}

	delete(k.functions, functionName)
	log.Printf("[KafkaManager] Kafka consumer unregistered for %s", functionName)
	return nil
}

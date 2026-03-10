package event

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/open-lambda/open-lambda/go/boss/lambdastore"
	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/lambda"
)

// KafkaManager manages Kafka consumption for all lambdas on a worker.
// It maintains a shared message cache and a KafkaFetcher. When triggers are
// registered for a lambda, a background loop automatically consumes messages
// and invokes the lambda.
type KafkaManager struct {
	triggerConfigs map[string][]common.KafkaTrigger      // lambdaName → trigger configs
	lambdaManager  *lambda.LambdaMgr
	fetcher        *KafkaFetcher
	offsets        map[string]map[string]map[int32]int64 // groupId → topic → partition → next offset
	stopChans      map[string]chan struct{}               // lambdaName → stop signal for consumption loop
	mu             sync.Mutex
}

// NewKafkaManager creates a KafkaManager with a shared message cache and fetcher.
func NewKafkaManager(lambdaManager *lambda.LambdaMgr) (*KafkaManager, error) {
	cacheSizeMb := common.Conf.Kafka_cache_size_mb
	if cacheSizeMb <= 0 {
		cacheSizeMb = 256
	}
	maxConcurrent := common.Conf.Kafka_max_concurrent_fetches
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	cache := NewMessageCache(int64(cacheSizeMb) * 1024 * 1024)

	manager := &KafkaManager{
		triggerConfigs: make(map[string][]common.KafkaTrigger),
		lambdaManager:  lambdaManager,
		fetcher:        NewKafkaFetcher(cache, maxConcurrent),
		offsets:        make(map[string]map[string]map[int32]int64),
		stopChans:      make(map[string]chan struct{}),
	}

	slog.Info("Kafka manager initialized",
		"cache_size_mb", cacheSizeMb,
		"max_concurrent_fetches", maxConcurrent)
	return manager, nil
}

// RegisterLambdaKafkaTriggers stores Kafka trigger configs for a lambda and
// starts a background consumption loop that automatically fetches messages
// and invokes the lambda.
func (km *KafkaManager) RegisterLambdaKafkaTriggers(lambdaName string, triggers []common.KafkaTrigger) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if len(triggers) == 0 {
		return nil
	}

	// Stop any existing consumption loop for this lambda
	if stopChan, ok := km.stopChans[lambdaName]; ok {
		close(stopChan)
		delete(km.stopChans, lambdaName)
	}

	// Store trigger configs with auto-generated group IDs
	stored := make([]common.KafkaTrigger, len(triggers))
	for i, trigger := range triggers {
		trigger.GroupId = fmt.Sprintf("lambda-%s", lambdaName)
		stored[i] = trigger
	}
	km.triggerConfigs[lambdaName] = stored

	// Start background consumption loop
	stopChan := make(chan struct{})
	km.stopChans[lambdaName] = stopChan
	go km.consumeLoop(lambdaName, stopChan)

	slog.Info("Started Kafka consumption for lambda",
		"lambda", lambdaName,
		"trigger_count", len(triggers))

	return nil
}

// getOffset returns the next offset to consume for a given group/topic/partition.
// Returns 0 if no offset has been tracked yet.
func (km *KafkaManager) getOffset(groupId, topic string, partition int32) int64 {
	if gm, ok := km.offsets[groupId]; ok {
		if tm, ok := gm[topic]; ok {
			if off, ok := tm[partition]; ok {
				return off
			}
		}
	}
	return 0
}

// setOffset stores the next offset to consume for a given group/topic/partition.
func (km *KafkaManager) setOffset(groupId, topic string, partition int32, offset int64) {
	if _, ok := km.offsets[groupId]; !ok {
		km.offsets[groupId] = make(map[string]map[int32]int64)
	}
	if _, ok := km.offsets[groupId][topic]; !ok {
		km.offsets[groupId][topic] = make(map[int32]int64)
	}
	km.offsets[groupId][topic][partition] = offset
}

// consumeLoop continuously consumes messages for a lambda until stopped.
// On each iteration it tries the cache, falls back to Kafka on miss,
// and invokes the lambda. Backs off when no messages are available.
func (km *KafkaManager) consumeLoop(lambdaName string, stopChan chan struct{}) {
	for {
		select {
		case <-stopChan:
			slog.Info("Stopping consumption loop", "lambda", lambdaName)
			return
		default:
		}

		consumed, err := km.consumeNext(lambdaName)
		if err != nil {
			slog.Error("Consumption error", "lambda", lambdaName, "error", err)
			select {
			case <-stopChan:
				return
			case <-time.After(100 * time.Millisecond):
			}
			continue
		}

		if !consumed {
			select {
			case <-stopChan:
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
}

// consumeNext tries to consume a single message for the lambda. It checks the
// cache first, fetches from Kafka on miss, caches the result, and invokes the
// lambda. Returns true if a message was consumed.
func (km *KafkaManager) consumeNext(lambdaName string) (bool, error) {
	t := common.T0("kafka-consume-next")
	defer t.T1()

	km.mu.Lock()
	triggers := km.triggerConfigs[lambdaName]
	km.mu.Unlock()

	for _, trigger := range triggers {
		groupId := trigger.GroupId

		for _, topic := range trigger.Topics {
			// TODO: support multi-partition consumption by discovering partition count.
			partition := int32(0)

			km.mu.Lock()
			offset := km.getOffset(groupId, topic, partition)
			km.mu.Unlock()

			fetchCtx, fetchCancel := context.WithTimeout(context.Background(), 15*time.Second)
			msg, err := km.fetcher.Get(fetchCtx, trigger.BootstrapServers, topic, partition, offset)
			fetchCancel()
			if err != nil {
				slog.Error("Failed to get message",
					"lambda", lambdaName,
					"topic", topic,
					"error", err)
				continue
			}
			if msg == nil {
				continue
			}

			// Invoke the lambda
			requestPath := fmt.Sprintf("/run/%s/", lambdaName)
			req, err := http.NewRequest("POST", requestPath, bytes.NewReader(msg.Value))
			if err != nil {
				return false, fmt.Errorf("failed to create request: %w", err)
			}
			req.RequestURI = requestPath

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Kafka-Topic", topic)
			req.Header.Set("X-Kafka-Partition", fmt.Sprintf("%d", partition))
			req.Header.Set("X-Kafka-Offset", fmt.Sprintf("%d", offset))
			req.Header.Set("X-Kafka-Group-Id", groupId)

			w := httptest.NewRecorder()
			lambdaFunc := km.lambdaManager.Get(lambdaName)
			lambdaFunc.Invoke(w, req)

			km.mu.Lock()
			km.setOffset(groupId, topic, partition, offset+1)
			km.mu.Unlock()

			slog.Info("Kafka message consumed and lambda invoked",
				"lambda", lambdaName,
				"topic", topic,
				"partition", partition,
				"offset", offset,
				"status", w.Code)

			return true, nil
		}
	}

	return false, nil
}

// UnregisterLambdaKafkaTriggers stops consumption and removes Kafka triggers for a lambda.
func (km *KafkaManager) UnregisterLambdaKafkaTriggers(lambdaName string) {
	km.mu.Lock()
	defer km.mu.Unlock()

	if stopChan, ok := km.stopChans[lambdaName]; ok {
		close(stopChan)
		delete(km.stopChans, lambdaName)
	}

	delete(km.triggerConfigs, lambdaName)

	slog.Info("Unregistered Kafka triggers for lambda", "lambda", lambdaName)
}

// cleanup shuts down all consumption loops.
func (km *KafkaManager) cleanup() {
	slog.Info("Shutting down Kafka manager")

	km.mu.Lock()
	defer km.mu.Unlock()

	for lambdaName, stopChan := range km.stopChans {
		close(stopChan)
		slog.Info("Stopped consumption loop", "lambda", lambdaName)
	}
	km.stopChans = make(map[string]chan struct{})
	km.triggerConfigs = make(map[string][]common.KafkaTrigger)

	slog.Info("Kafka manager shutdown complete")
}

// HandleKafkaRegister handles Kafka consumer registration/unregistration for lambdas.
func HandleKafkaRegister(kafkaManager *KafkaManager, lambdaStore *lambdastore.LambdaStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lambdaName := strings.TrimPrefix(r.URL.Path, "/kafka/register/")
		if lambdaName == "" {
			http.Error(w, "lambda name required", http.StatusBadRequest)
			return
		}

		type Response struct {
			Status  string `json:"status"`
			Lambda  string `json:"lambda"`
			Message string `json:"message"`
		}

		switch r.Method {
		case "POST":
			config, err := lambdaStore.GetConfig(lambdaName)
			if err != nil {
				slog.Error("Failed to load lambda config for Kafka registration",
					"lambda", lambdaName,
					"error", err)
				http.Error(w, fmt.Sprintf("failed to load lambda config: %v", err), http.StatusNotFound)
				return
			}

			if config == nil || len(config.Triggers.Kafka) == 0 {
				http.Error(w, "lambda has no Kafka triggers", http.StatusBadRequest)
				return
			}

			err = kafkaManager.RegisterLambdaKafkaTriggers(lambdaName, config.Triggers.Kafka)
			if err != nil {
				slog.Error("Failed to register Kafka triggers",
					"lambda", lambdaName,
					"error", err)
				http.Error(w, fmt.Sprintf("failed to register Kafka triggers: %v", err), http.StatusInternalServerError)
				return
			}

			slog.Info("Registered Kafka triggers via API",
				"lambda", lambdaName,
				"triggers", len(config.Triggers.Kafka))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Response{
				Status:  "success",
				Lambda:  lambdaName,
				Message: fmt.Sprintf("Kafka triggers registered for %d trigger(s)", len(config.Triggers.Kafka)),
			})

		case "DELETE":
			kafkaManager.UnregisterLambdaKafkaTriggers(lambdaName)

			slog.Info("Unregistered Kafka triggers via API", "lambda", lambdaName)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Response{
				Status:  "success",
				Lambda:  lambdaName,
				Message: "Kafka triggers unregistered",
			})

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

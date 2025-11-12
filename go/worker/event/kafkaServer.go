package event

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/open-lambda/open-lambda/go/boss/lambdastore"
	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/lambda"
	"github.com/twmb/franz-go/pkg/kgo"
)

type KafkaClient interface {
	PollFetches(context.Context) kgo.Fetches
	Close()
}

// TriggerKey uniquely identifies a Kafka trigger configuration
type TriggerKey string

// ComputeTriggerKey creates a deterministic key from trigger config
func ComputeTriggerKey(trigger *common.KafkaTrigger) TriggerKey {
	// Sort brokers and topics for consistency
	brokers := make([]string, len(trigger.BootstrapServers))
	copy(brokers, trigger.BootstrapServers)
	sort.Strings(brokers)

	topics := make([]string, len(trigger.Topics))
	copy(topics, trigger.Topics)
	sort.Strings(topics)

	// Create canonical key: brokers|topics|offset_reset
	key := fmt.Sprintf("%s|%s|%s",
		strings.Join(brokers, ","),
		strings.Join(topics, ","),
		trigger.AutoOffsetReset)

	return TriggerKey(key)
}

// LambdaKafkaConsumer manages Kafka consumption for one or more lambda functions
// with identical trigger configurations
type LambdaKafkaConsumer struct {
	triggerKey    TriggerKey                   // Unique identifier for this trigger config
	kafkaTrigger  *common.KafkaTrigger         // Trigger configuration
	client        KafkaClient                  // kgo.client implements the KafkaClient interface
	lambdaManager *lambda.LambdaMgr            // Reference to lambda manager for direct calls
	lambdas       map[string]struct{}          // Set of lambda names using this consumer
	lambdasMu     sync.RWMutex                 // Protects lambdas map
	stopChan      chan struct{}                // Shutdown signal for this consumer
}

// KafkaManager manages Kafka consumers shared across lambda functions
type KafkaManager struct {
	consumers      map[TriggerKey]*LambdaKafkaConsumer // triggerKey -> consumer
	lambdaTriggers map[string][]TriggerKey             // lambdaName -> triggerKeys
	lambdaManager  *lambda.LambdaMgr                   // Reference to lambda manager
	mu             sync.Mutex                          // Protects consumers and lambdaTriggers maps
}

// Helper methods for LambdaKafkaConsumer

// addLambda adds a lambda to this consumer
func (lkc *LambdaKafkaConsumer) addLambda(lambdaName string) {
	lkc.lambdasMu.Lock()
	defer lkc.lambdasMu.Unlock()
	lkc.lambdas[lambdaName] = struct{}{}
}

// removeLambda removes a lambda from this consumer
func (lkc *LambdaKafkaConsumer) removeLambda(lambdaName string) {
	lkc.lambdasMu.Lock()
	defer lkc.lambdasMu.Unlock()
	delete(lkc.lambdas, lambdaName)
}

// getLambdaCount returns the number of lambdas using this consumer
func (lkc *LambdaKafkaConsumer) getLambdaCount() int {
	lkc.lambdasMu.RLock()
	defer lkc.lambdasMu.RUnlock()
	return len(lkc.lambdas)
}

// getLambdaNames returns a snapshot of lambda names (thread-safe)
func (lkc *LambdaKafkaConsumer) getLambdaNames() []string {
	lkc.lambdasMu.RLock()
	defer lkc.lambdasMu.RUnlock()

	names := make([]string, 0, len(lkc.lambdas))
	for name := range lkc.lambdas {
		names = append(names, name)
	}
	return names
}

// newLambdaKafkaConsumer creates a new Kafka consumer for a trigger configuration
func (km *KafkaManager) newLambdaKafkaConsumer(triggerKey TriggerKey, trigger *common.KafkaTrigger) (*LambdaKafkaConsumer, error) {
	// Validate that we have brokers and topics
	if len(trigger.BootstrapServers) == 0 {
		return nil, fmt.Errorf("no bootstrap servers configured for trigger %s", triggerKey)
	}
	if len(trigger.Topics) == 0 {
		return nil, fmt.Errorf("no topics configured for trigger %s", triggerKey)
	}

	// Setup kgo client options
	opts := []kgo.Opt{
		kgo.SeedBrokers(trigger.BootstrapServers...),
		kgo.ConsumerGroup(trigger.GroupId),
		kgo.ConsumeTopics(trigger.Topics...),
		kgo.SessionTimeout(10 * time.Second),
		kgo.HeartbeatInterval(3 * time.Second),
	}

	// Use trigger-specific offset reset or default to latest
	if trigger.AutoOffsetReset == "earliest" {
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	} else {
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
	}

	// Create kgo client
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client for trigger %s: %w", triggerKey, err)
	}

	return &LambdaKafkaConsumer{
		triggerKey:    triggerKey,
		kafkaTrigger:  trigger,
		client:        client,
		lambdaManager: km.lambdaManager,
		lambdas:       make(map[string]struct{}),
		stopChan:      make(chan struct{}),
	}, nil
}

// NewKafkaManager creates and configures a new Kafka manager
func NewKafkaManager(lambdaManager *lambda.LambdaMgr) (*KafkaManager, error) {
	manager := &KafkaManager{
		consumers:      make(map[TriggerKey]*LambdaKafkaConsumer),
		lambdaTriggers: make(map[string][]TriggerKey),
		lambdaManager:  lambdaManager,
	}

	slog.Info("Kafka manager initialized with consumer sharing enabled")
	return manager, nil
}

// StartConsuming starts consuming messages for Kafka triggers
func (lkc *LambdaKafkaConsumer) StartConsuming() {
	slog.Info("Starting Kafka consumer",
		"trigger_key", lkc.triggerKey,
		"topics", lkc.kafkaTrigger.Topics,
		"brokers", lkc.kafkaTrigger.BootstrapServers,
		"group_id", lkc.kafkaTrigger.GroupId,
		"initial_lambdas", lkc.getLambdaCount())

	// Start consuming loop
	go lkc.consumeLoop()
}

// consumeLoop handles Kafka message consumption using kgo polling
func (lkc *LambdaKafkaConsumer) consumeLoop() {
	for {
		select {
		case <-lkc.stopChan:
			slog.Info("Stopping Kafka consumer", "trigger_key", lkc.triggerKey)
			return
		default:
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			fetches := lkc.client.PollFetches(ctx)
			cancel()

			if errs := fetches.Errors(); len(errs) > 0 {
				for _, err := range errs {
					// The franz-go package uses context.DeadlineExceeded to signal that the
					// timeout has expired with no new messages. This is expected
					// behaviour during idle periods and should not be logged.
					if errors.Is(err.Err, context.DeadlineExceeded) {
						continue
					}

					// TODO: Surface Kafka consumer errors to lambda developers by invoking an error
					// handler lambda function. Could allow lambdas to specify an onError callback in
					// ol.yaml that gets invoked with error details.
					slog.Warn("Kafka fetch error",
						"trigger_key", lkc.triggerKey,
						"error", err)
				}
				continue
			}

			// Process each record
			fetches.EachRecord(func(record *kgo.Record) {
				slog.Info("Received Kafka message",
					"trigger_key", lkc.triggerKey,
					"topic", record.Topic,
					"partition", record.Partition,
					"offset", record.Offset,
					"size", len(record.Value))
				lkc.processMessage(record)
			})
		}
	}
}

// processMessage handles a single Kafka message by invoking all registered lambda functions
func (lkc *LambdaKafkaConsumer) processMessage(record *kgo.Record) {
	t := common.T0("kafka-message-processing")
	defer t.T1()

	// Get snapshot of lambdas (thread-safe)
	lambdaNames := lkc.getLambdaNames()

	if len(lambdaNames) == 0 {
		slog.Warn("No lambdas registered for consumer", "trigger_key", lkc.triggerKey)
		return
	}

	slog.Info("Processing Kafka message for multiple lambdas",
		"trigger_key", lkc.triggerKey,
		"topic", record.Topic,
		"partition", record.Partition,
		"offset", record.Offset,
		"lambda_count", len(lambdaNames))

	// Invoke all lambdas in parallel
	var wg sync.WaitGroup
	for _, lambdaName := range lambdaNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			lkc.invokeLambda(name, record)
		}(lambdaName)
	}

	wg.Wait()
}

// invokeLambda invokes a single lambda function with a Kafka message
func (lkc *LambdaKafkaConsumer) invokeLambda(lambdaName string, record *kgo.Record) {
	// Create synthetic HTTP request from Kafka message
	req, err := http.NewRequest("POST", "/", bytes.NewReader(record.Value))
	if err != nil {
		slog.Error("Failed to create request for lambda invocation",
			"lambda", lambdaName,
			"error", err,
			"topic", record.Topic)
		return
	}

	// Set headers with Kafka metadata (The X- prefix indicates a custom non-standard header)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Kafka-Topic", record.Topic)
	req.Header.Set("X-Kafka-Partition", fmt.Sprintf("%d", record.Partition))
	req.Header.Set("X-Kafka-Offset", fmt.Sprintf("%d", record.Offset))
	req.Header.Set("X-Kafka-Group-Id", lkc.kafkaTrigger.GroupId)

	// Create response recorder to capture lambda output.
	// TODO: Capture and log the lambda response body using httptest's response recorder
	// for kafka triggered lambda invocations.
	w := httptest.NewRecorder()

	// Get lambda function and invoke directly
	lambdaFunc := lkc.lambdaManager.Get(lambdaName)
	lambdaFunc.Invoke(w, req)

	// Log the result
	slog.Info("Lambda invoked from Kafka consumer",
		"lambda", lambdaName,
		"trigger_key", lkc.triggerKey,
		"topic", record.Topic,
		"offset", record.Offset,
		"status", w.Code)
}

// cleanup closes the kgo client
func (lkc *LambdaKafkaConsumer) cleanup() {
	slog.Info("Shutting down Kafka consumer", "trigger_key", lkc.triggerKey)

	// Signal all goroutines to stop
	close(lkc.stopChan)

	// Close kgo client
	if lkc.client != nil {
		lkc.client.Close()
	}

	slog.Info("Kafka consumer shutdown complete", "trigger_key", lkc.triggerKey)
}

// RegisterLambdaKafkaTriggers registers Kafka triggers for a lambda function
func (km *KafkaManager) RegisterLambdaKafkaTriggers(lambdaName string, triggers []common.KafkaTrigger) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if len(triggers) == 0 {
		return nil // No Kafka triggers for this lambda
	}

	// Step 1: Unregister old triggers for this lambda (if any)
	km.unregisterLambdaInternal(lambdaName)

	var registeredKeys []TriggerKey

	// Step 2: Process each trigger
	for _, trigger := range triggers {
		// Compute trigger identity
		triggerKey := ComputeTriggerKey(&trigger)

		// Step 3: Check if consumer already exists
		consumer, exists := km.consumers[triggerKey]

		if exists {
			// REUSE: Add lambda to existing consumer
			consumer.addLambda(lambdaName)
			slog.Info("Reusing existing Kafka consumer",
				"lambda", lambdaName,
				"trigger_key", triggerKey,
				"total_lambdas", consumer.getLambdaCount())
		} else {
			// CREATE: New consumer needed
			trigger.GroupId = string(triggerKey) // Use trigger key as group ID

			consumer, err := km.newLambdaKafkaConsumer(triggerKey, &trigger)
			if err != nil {
				slog.Error("Failed to create Kafka consumer",
					"lambda", lambdaName,
					"error", err)
				continue
			}

			consumer.addLambda(lambdaName) // Add first lambda
			km.consumers[triggerKey] = consumer

			// Start consuming
			go consumer.StartConsuming()

			slog.Info("Created new Kafka consumer",
				"lambda", lambdaName,
				"trigger_key", triggerKey,
				"topics", trigger.Topics,
				"brokers", trigger.BootstrapServers,
				"group_id", trigger.GroupId)
		}

		registeredKeys = append(registeredKeys, triggerKey)
	}

	// Step 4: Track reverse mapping
	km.lambdaTriggers[lambdaName] = registeredKeys

	return nil
}

// UnregisterLambdaKafkaTriggers removes Kafka triggers for a lambda function
func (km *KafkaManager) UnregisterLambdaKafkaTriggers(lambdaName string) {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.unregisterLambdaInternal(lambdaName)
}

// unregisterLambdaInternal removes lambda from all consumers (must be called with lock held)
func (km *KafkaManager) unregisterLambdaInternal(lambdaName string) {
	// Find triggers this lambda was using
	triggerKeys, exists := km.lambdaTriggers[lambdaName]
	if !exists {
		return
	}

	// Remove lambda from each consumer
	for _, triggerKey := range triggerKeys {
		consumer, exists := km.consumers[triggerKey]
		if !exists {
			continue
		}

		consumer.removeLambda(lambdaName)

		// Check if consumer is now unused
		if consumer.getLambdaCount() == 0 {
			slog.Info("Shutting down unused Kafka consumer",
				"trigger_key", triggerKey)

			consumer.cleanup()
			delete(km.consumers, triggerKey)
		} else {
			slog.Info("Lambda removed from shared consumer",
				"lambda", lambdaName,
				"trigger_key", triggerKey,
				"remaining_lambdas", consumer.getLambdaCount())
		}
	}

	delete(km.lambdaTriggers, lambdaName)
}

// cleanup closes all lambda consumers
func (km *KafkaManager) cleanup() {
	slog.Info("Shutting down Kafka manager")

	km.mu.Lock()
	defer km.mu.Unlock()

	// Close all consumers
	for triggerKey, consumer := range km.consumers {
		lambdaCount := consumer.getLambdaCount()
		consumer.cleanup()
		slog.Info("Cleaned up Kafka consumer",
			"trigger_key", triggerKey,
			"lambdas_affected", lambdaCount)
	}

	// Clear the maps
	km.consumers = make(map[TriggerKey]*LambdaKafkaConsumer)
	km.lambdaTriggers = make(map[string][]TriggerKey)

	slog.Info("Kafka manager shutdown complete")
}

// HandleKafkaRegister handles Kafka consumer registration/unregistration for lambdas
func HandleKafkaRegister(kafkaManager *KafkaManager, lambdaStore *lambdastore.LambdaStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract lambda name
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
			// Read lambda config from registry
			config, err := lambdaStore.GetConfig(lambdaName)
			if err != nil {
				slog.Error("Failed to load lambda config for Kafka registration",
					"lambda", lambdaName,
					"error", err)
				http.Error(w, fmt.Sprintf("failed to load lambda config: %v", err), http.StatusNotFound)
				return
			}

			// Check if lambda has Kafka triggers
			if config == nil || len(config.Triggers.Kafka) == 0 {
				http.Error(w, "lambda has no Kafka triggers", http.StatusBadRequest)
				return
			}

			// Register Kafka triggers
			err = kafkaManager.RegisterLambdaKafkaTriggers(lambdaName, config.Triggers.Kafka)
			if err != nil {
				slog.Error("Failed to register Kafka triggers",
					"lambda", lambdaName,
					"error", err)
				http.Error(w, fmt.Sprintf("failed to register Kafka triggers: %v", err), http.StatusInternalServerError)
				return
			}

			slog.Info("Registered Kafka consumers via API",
				"lambda", lambdaName,
				"triggers", len(config.Triggers.Kafka))

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Response{
				Status:  "success",
				Lambda:  lambdaName,
				Message: fmt.Sprintf("Kafka consumers registered for %d trigger(s)", len(config.Triggers.Kafka)),
			})

		case "DELETE":
			// Unregister Kafka triggers
			kafkaManager.UnregisterLambdaKafkaTriggers(lambdaName)

			slog.Info("Unregistered Kafka consumers via API", "lambda", lambdaName)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(Response{
				Status:  "success",
				Lambda:  lambdaName,
				Message: "Kafka consumers unregistered",
			})

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

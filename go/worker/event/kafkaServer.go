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

// LambdaKafkaConsumer manages Kafka consumption for a specific lambda function
type LambdaKafkaConsumer struct {
	consumerName  string // Unique name for this consumer
	lambdaName    string // lambda function name
	kafkaTrigger  *common.KafkaTrigger
	client        KafkaClient       // kgo.client implements the KafkaClient interface
	lambdaManager *lambda.LambdaMgr // Reference to lambda manager for direct calls
	stopChan      chan struct{}     // Shutdown signal for this consumer
	// When this channel is closed, the goroutine for the consumer exits
}

// KafkaManager manages multiple lambda-specific Kafka consumers
type KafkaManager struct {
	lambdaConsumers map[string]*LambdaKafkaConsumer // lambdaName -> consumer
	lambdaManager   *lambda.LambdaMgr               // Reference to lambda manager
	mu              sync.Mutex                      // Protects lambdaConsumers map
}

// newLambdaKafkaConsumer creates a new Kafka consumer for a specific lambda function
func (km *KafkaManager) newLambdaKafkaConsumer(consumerName string, lambdaName string, trigger *common.KafkaTrigger) (*LambdaKafkaConsumer, error) {
	// Validate that we have brokers and topics
	if len(trigger.BootstrapServers) == 0 {
		return nil, fmt.Errorf("no bootstrap servers configured for lambda %s", lambdaName)
	}
	if len(trigger.Topics) == 0 {
		return nil, fmt.Errorf("no topics configured for lambda %s", lambdaName)
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
		return nil, fmt.Errorf("failed to create Kafka client for lambda %s: %w", lambdaName, err)
	}

	return &LambdaKafkaConsumer{
		consumerName:  consumerName,
		lambdaName:    lambdaName,
		kafkaTrigger:  trigger,
		client:        client,
		lambdaManager: km.lambdaManager,
		stopChan:      make(chan struct{}),
	}, nil
}

// NewKafkaManager creates and configures a new Kafka manager
func NewKafkaManager(lambdaManager *lambda.LambdaMgr) (*KafkaManager, error) {
	manager := &KafkaManager{
		lambdaConsumers: make(map[string]*LambdaKafkaConsumer),
		lambdaManager:   lambdaManager,
	}

	slog.Info("Kafka manager initialized")
	return manager, nil
}

// StartConsuming starts consuming messages for this lambda's Kafka triggers
func (lkc *LambdaKafkaConsumer) StartConsuming() {
	slog.Info("Starting Kafka consumer for lambda",
		"consumer", lkc.consumerName,
		"lambda", lkc.lambdaName,
		"topics", lkc.kafkaTrigger.Topics,
		"brokers", lkc.kafkaTrigger.BootstrapServers,
		"group_id", lkc.kafkaTrigger.GroupId)

	// Start consuming loop
	go lkc.consumeLoop()
}

// consumeLoop handles Kafka message consumption using kgo polling
func (lkc *LambdaKafkaConsumer) consumeLoop() {
	for {
		select {
		case <-lkc.stopChan:
			slog.Info("Stopping Kafka consumer for lambda", "lambda", lkc.lambdaName)
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
						"lambda", lkc.lambdaName,
						"error", err)
				}
				continue
			}

			// Process each record
			fetches.EachRecord(func(record *kgo.Record) {
				slog.Info("Received Kafka message for lambda",
					"consumer", lkc.consumerName,
					"lambda", lkc.lambdaName,
					"topic", record.Topic,
					"partition", record.Partition,
					"offset", record.Offset,
					"size", len(record.Value))
				lkc.processMessage(record)
			})
		}
	}
}

// processMessage handles a single Kafka message by invoking the lambda function directly
func (lkc *LambdaKafkaConsumer) processMessage(record *kgo.Record) {
	t := common.T0("kafka-message-processing")
	defer t.T1()

	// Create synthetic HTTP request from Kafka message
	req, err := http.NewRequest("POST", "/", bytes.NewReader(record.Value))
	if err != nil {
		slog.Error("Failed to create request for lambda invocation",
			"lambda", lkc.lambdaName,
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
	lambdaFunc := lkc.lambdaManager.Get(lkc.lambdaName)
	lambdaFunc.Invoke(w, req)

	// Log the result
	slog.Info("Kafka message processed via direct invocation",
		"consumer", lkc.consumerName,
		"lambda", lkc.lambdaName,
		"topic", record.Topic,
		"partition", record.Partition,
		"offset", record.Offset,
		"status", w.Code)
}

// cleanup closes the kgo client
func (lkc *LambdaKafkaConsumer) cleanup() {
	slog.Info("Shutting down Kafka consumer for lambda", "lambda", lkc.lambdaName)

	// Signal all goroutines to stop
	close(lkc.stopChan)

	// Close kgo client
	if lkc.client != nil {
		lkc.client.Close()
	}

	slog.Info("Kafka consumer shutdown complete for lambda", "lambda", lkc.lambdaName)
}

// RegisterLambdaKafkaTriggers registers Kafka triggers for a lambda function
func (km *KafkaManager) RegisterLambdaKafkaTriggers(lambdaName string, triggers []common.KafkaTrigger) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if len(triggers) == 0 {
		return nil // No Kafka triggers for this lambda
	}

	// If lambda already has consumers, clean them up first
	for consumerName, consumer := range km.lambdaConsumers {
		if strings.HasPrefix(consumerName, lambdaName+"-") {
			consumer.cleanup()
			delete(km.lambdaConsumers, consumerName)
			slog.Info("Cleaned up existing Kafka consumer for lambda", "lambda", lambdaName, "consumer", consumerName)
		}
	}

	// Create consumers for each Kafka trigger
	for i, trigger := range triggers {
		trigger.GroupId = fmt.Sprintf("lambda-%s", lambdaName)
		consumerName := fmt.Sprintf("%s-%d", lambdaName, i)
		consumer, err := km.newLambdaKafkaConsumer(consumerName, lambdaName, &trigger)
		if err != nil {
			slog.Error("Failed to create Kafka consumer for lambda",
				"lambda", lambdaName,
				"trigger_index", i,
				"error", err)
			continue
		}

		km.lambdaConsumers[consumerName] = consumer

		// Start consuming in background
		go func(c *LambdaKafkaConsumer) {
			c.StartConsuming()
		}(consumer)

		slog.Info("Registered Kafka trigger for lambda",
			"lambda", lambdaName,
			"topics", trigger.Topics,
			"brokers", trigger.BootstrapServers,
			"group_id", trigger.GroupId)
	}

	return nil
}

// UnregisterLambdaKafkaTriggers removes Kafka triggers for a lambda function
func (km *KafkaManager) UnregisterLambdaKafkaTriggers(lambdaName string) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Find and cleanup all consumers for this lambda
	for consumerName, consumer := range km.lambdaConsumers {
		if strings.HasPrefix(consumerName, lambdaName+"-") {
			consumer.cleanup()
			delete(km.lambdaConsumers, consumerName)
			slog.Info("Unregistered Kafka consumer for lambda", "lambda", lambdaName)
		}
	}
}

// cleanup closes all lambda consumers
func (km *KafkaManager) cleanup() {
	slog.Info("Shutting down Kafka manager")

	km.mu.Lock()
	defer km.mu.Unlock()

	// Close all lambda consumers
	for lambdaName, consumer := range km.lambdaConsumers {
		consumer.cleanup()
		slog.Info("Cleaned up Kafka consumer for lambda", "lambda", lambdaName)
	}

	// Clear the map
	km.lambdaConsumers = make(map[string]*LambdaKafkaConsumer)

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

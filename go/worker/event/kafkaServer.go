package event

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

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
}

// KafkaServer manages multiple lambda-specific Kafka consumers
type KafkaServer struct {
	lambdaConsumers map[string]*LambdaKafkaConsumer // lambdaName -> consumer
	lambdaManager   *lambda.LambdaMgr               // Reference to lambda manager
	mu              sync.Mutex                      // Protects lambdaConsumers map
	stopChan        chan struct{}                   // shutdown signal for all consumers in the worker
}

// NewLambdaKafkaConsumer creates a new Kafka consumer for a specific lambda function
func NewLambdaKafkaConsumer(consumerName string, lambdaName string, trigger *common.KafkaTrigger, lambdaManager *lambda.LambdaMgr) (*LambdaKafkaConsumer, error) {
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
		lambdaManager: lambdaManager,
		stopChan:      make(chan struct{}),
	}, nil
}

// NewKafkaServer creates and configures a new Kafka server
func NewKafkaServer(lambdaManager *lambda.LambdaMgr) (*KafkaServer, error) {
	server := &KafkaServer{
		lambdaConsumers: make(map[string]*LambdaKafkaConsumer),
		lambdaManager:   lambdaManager,
		stopChan:        make(chan struct{}),
	}

	slog.Info("Kafka server initialized")
	return server, nil
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

	// Block until shutdown
	<-lkc.stopChan
}

// consumeLoop handles Kafka message consumption using kgo polling
func (lkc *LambdaKafkaConsumer) consumeLoop() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic in consume loop",
				"lambda", lkc.lambdaName,
				"error", r)
		}
	}()

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
					if errors.Is(err.Err, context.DeadlineExceeded) {
						continue
					}
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
				go lkc.processMessage(record)
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

	// Set headers with Kafka metadata
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Kafka-Topic", record.Topic)
	req.Header.Set("X-Kafka-Partition", fmt.Sprintf("%d", record.Partition))
	req.Header.Set("X-Kafka-Offset", fmt.Sprintf("%d", record.Offset))
	req.Header.Set("X-Kafka-Group-Id", lkc.kafkaTrigger.GroupId)

	// Create response recorder to capture lambda output
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
func (ks *KafkaServer) RegisterLambdaKafkaTriggers(lambdaName string, triggers []common.KafkaTrigger) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	if len(triggers) == 0 {
		return nil // No Kafka triggers for this lambda
	}

	// If lambda already has consumers, clean them up first
	for consumerName, consumer := range ks.lambdaConsumers {
		if strings.HasPrefix(consumerName, lambdaName+"-") {
			consumer.cleanup()
			delete(ks.lambdaConsumers, consumerName)
			slog.Info("Cleaned up existing Kafka consumer for lambda", "lambda", lambdaName, "consumer", consumerName)
		}
	}

	// Create consumers for each Kafka trigger
	for i, trigger := range triggers {
		consumerName := fmt.Sprintf("%s-%d", lambdaName, i)
		consumer, err := NewLambdaKafkaConsumer(consumerName, lambdaName, &trigger, ks.lambdaManager)
		if err != nil {
			slog.Error("Failed to create Kafka consumer for lambda",
				"lambda", lambdaName,
				"trigger_index", i,
				"error", err)
			continue
		}

		ks.lambdaConsumers[consumerName] = consumer

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
func (ks *KafkaServer) UnregisterLambdaKafkaTriggers(lambdaName string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	// Find and cleanup all consumers for this lambda
	for consumerName, consumer := range ks.lambdaConsumers {
		if strings.HasPrefix(consumerName, lambdaName+"-") {
			consumer.cleanup()
			delete(ks.lambdaConsumers, consumerName)
			slog.Info("Unregistered Kafka consumer for lambda", "lambda", lambdaName)
		}
	}
}

// StartConsuming starts the Kafka server (all lambda consumers are managed separately)
func (ks *KafkaServer) StartConsuming() {
	slog.Info("Kafka server ready to manage lambda consumers")

	// Block until shutdown
	<-ks.stopChan
}

// cleanup closes all lambda consumers
func (ks *KafkaServer) cleanup() {
	slog.Info("Shutting down Kafka server")

	// Signal all goroutines to stop
	close(ks.stopChan)

	ks.mu.Lock()
	defer ks.mu.Unlock()

	// Close all lambda consumers
	for lambdaName, consumer := range ks.lambdaConsumers {
		consumer.cleanup()
		slog.Info("Cleaned up Kafka consumer for lambda", "lambda", lambdaName)
	}

	// Clear the map
	ks.lambdaConsumers = make(map[string]*LambdaKafkaConsumer)

	slog.Info("Kafka server shutdown complete")
}

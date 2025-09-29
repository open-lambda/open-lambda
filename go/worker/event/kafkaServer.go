package event

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/Shopify/sarama"
	"github.com/open-lambda/open-lambda/go/common"
	"github.com/open-lambda/open-lambda/go/worker/lambda"
)


// LambdaKafkaConsumer manages Kafka consumption for a specific lambda function
type LambdaKafkaConsumer struct {
	consumerName     string // Unique name for this consumer (may include index suffix)
	lambdaName       string // Actual lambda function name for invocation
	kafkaTrigger     *common.KafkaTrigger
	consumer         sarama.Consumer
	partitionConsumers []sarama.PartitionConsumer
	lambdaManager    *lambda.LambdaMgr // Reference to lambda manager for direct calls
	stopChan         chan struct{}
}

// KafkaServer manages multiple lambda-specific Kafka consumers
type KafkaServer struct {
	lambdaConsumers map[string]*LambdaKafkaConsumer // lambdaName -> consumer
	lambdaManager   *lambda.LambdaMgr          // Reference to lambda manager
	mu              sync.RWMutex
	stopChan        chan struct{}
}

// NewLambdaKafkaConsumer creates a new Kafka consumer for a specific lambda function
func NewLambdaKafkaConsumer(consumerName string, lambdaName string, trigger *common.KafkaTrigger, lambdaManager *lambda.LambdaMgr) (*LambdaKafkaConsumer, error) {
	// Setup Kafka consumer configuration
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	// Use trigger-specific offset reset or default to latest
	if trigger.AutoOffsetReset == "earliest" {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	} else {
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	}

	config.Consumer.Group.Session.Timeout = 10 * time.Second
	config.Consumer.Group.Heartbeat.Interval = 3 * time.Second
	config.Version = sarama.V2_6_0_0

	// Create consumer
	consumer, err := sarama.NewConsumer(trigger.BootstrapServers, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka consumer for lambda %s: %w", lambdaName, err)
	}

	return &LambdaKafkaConsumer{
		consumerName:  consumerName,
		lambdaName:    lambdaName,
		kafkaTrigger:  trigger,
		consumer:      consumer,
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
func (lkc *LambdaKafkaConsumer) StartConsuming() error {
	slog.Info("Starting Kafka consumer for lambda",
		"consumer", lkc.consumerName,
		"lambda", lkc.lambdaName,
		"topics", lkc.kafkaTrigger.Topics,
		"brokers", lkc.kafkaTrigger.BootstrapServers,
		"group_id", lkc.kafkaTrigger.GroupId)

	// Subscribe to all topics for this lambda
	for _, topic := range lkc.kafkaTrigger.Topics {
		if err := lkc.subscribeToTopic(topic); err != nil {
			return fmt.Errorf("failed to subscribe lambda %s to topic %s: %w", lkc.lambdaName, topic, err)
		}
	}

	slog.Info("Kafka consumer started for lambda", "consumer", lkc.consumerName, "lambda", lkc.lambdaName)

	// Block until shutdown
	<-lkc.stopChan
	return nil
}

// subscribeToTopic subscribes to a specific Kafka topic for this lambda
func (lkc *LambdaKafkaConsumer) subscribeToTopic(topic string) error {
	partitionList, err := lkc.consumer.Partitions(topic)
	if err != nil {
		return fmt.Errorf("failed to get partitions for topic %s: %w", topic, err)
	}

	for _, partition := range partitionList {
		pc, err := lkc.consumer.ConsumePartition(topic, partition, sarama.OffsetOldest)
		if err != nil {
			return fmt.Errorf("failed to start partition consumer for %s:%d: %w",
				topic, partition, err)
		}

		lkc.partitionConsumers = append(lkc.partitionConsumers, pc)

		// Start goroutine to handle messages from this partition
		go lkc.consumePartition(pc)
	}

	slog.Info("Lambda subscribed to topic",
		"consumer", lkc.consumerName,
		"lambda", lkc.lambdaName,
		"topic", topic,
		"partitions", len(partitionList))
	return nil
}

// consumePartition handles messages from a specific partition for this lambda
func (lkc *LambdaKafkaConsumer) consumePartition(pc sarama.PartitionConsumer) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic in partition consumer",
				"lambda", lkc.lambdaName,
				"error", r)
		}
		pc.Close()
	}()

	for {
		select {
		case message := <-pc.Messages():
			if message != nil {
				slog.Info("Received Kafka message for lambda",
					"consumer", lkc.consumerName,
					"lambda", lkc.lambdaName,
					"topic", message.Topic,
					"partition", message.Partition,
					"offset", message.Offset,
					"size", len(message.Value))
				go lkc.processMessage(message)
			}
		case err := <-pc.Errors():
			if err != nil {
				slog.Error("Partition consumer error",
					"lambda", lkc.lambdaName,
					"error", err)
			}
		case <-lkc.stopChan:
			slog.Info("Stopping partition consumer for lambda", "lambda", lkc.lambdaName)
			return
		}
	}
}

// processMessage handles a single Kafka message by invoking the lambda function directly
func (lkc *LambdaKafkaConsumer) processMessage(msg *sarama.ConsumerMessage) {
	t := common.T0("kafka-message-processing")
	defer t.T1()

	// Create synthetic HTTP request from Kafka message
	req, err := http.NewRequest("POST", "/", bytes.NewReader(msg.Value))
	if err != nil {
		slog.Error("Failed to create request for lambda invocation",
			"lambda", lkc.lambdaName,
			"error", err,
			"topic", msg.Topic)
		return
	}

	// Set headers with Kafka metadata
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Kafka-Topic", msg.Topic)
	req.Header.Set("X-Kafka-Partition", fmt.Sprintf("%d", msg.Partition))
	req.Header.Set("X-Kafka-Offset", fmt.Sprintf("%d", msg.Offset))
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
		"topic", msg.Topic,
		"partition", msg.Partition,
		"offset", msg.Offset,
		"status", w.Code)
}


// cleanup closes all partition consumers and the main consumer
func (lkc *LambdaKafkaConsumer) cleanup() {
	slog.Info("Shutting down Kafka consumer for lambda", "lambda", lkc.lambdaName)

	// Signal all goroutines to stop
	close(lkc.stopChan)

	// Close partition consumers
	for _, pc := range lkc.partitionConsumers {
		if err := pc.Close(); err != nil {
			slog.Error("Error closing partition consumer",
				"lambda", lkc.lambdaName,
				"error", err)
		}
	}

	// Close main consumer
	if lkc.consumer != nil {
		if err := lkc.consumer.Close(); err != nil {
			slog.Error("Error closing Kafka consumer",
				"lambda", lkc.lambdaName,
				"error", err)
		}
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
	if existingConsumer, exists := ks.lambdaConsumers[lambdaName]; exists {
		existingConsumer.cleanup()
		delete(ks.lambdaConsumers, lambdaName)
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
			if err := c.StartConsuming(); err != nil {
				slog.Error("Kafka consumer error for lambda",
					"lambda", lambdaName,
					"error", err)
			}
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
func (ks *KafkaServer) StartConsuming() error {
	slog.Info("Kafka server ready to manage lambda consumers")

	// Block until shutdown
	<-ks.stopChan
	return nil
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
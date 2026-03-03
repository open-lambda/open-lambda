package event

import (
	"bytes"
	"context"
	"encoding/base64"
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

// LambdaKafkaConsumer manages Kafka consumption for a specific lambda function.
// Retained for backwards compatibility with the push-based consumption model.
type LambdaKafkaConsumer struct {
	consumerName  string // Unique name for this consumer
	lambdaName    string // lambda function name
	kafkaTrigger  *common.KafkaTrigger
	client        KafkaClient       // kgo.client implements the KafkaClient interface
	lambdaManager *lambda.LambdaMgr // Reference to lambda manager for direct calls
	stopChan      chan struct{}      // Shutdown signal for this consumer
}

// KafkaManager manages Kafka consumption for all lambdas on a worker.
// It maintains a shared message cache and a KafkaFetcher for the pull-based
// consumption model, as well as legacy push-based consumers.
type KafkaManager struct {
	lambdaConsumers map[string]*LambdaKafkaConsumer // push-model consumers (legacy)
	triggerConfigs  map[string][]common.KafkaTrigger // lambdaName → trigger configs for pull model
	lambdaManager   *lambda.LambdaMgr
	cache           *MessageCache
	fetcher         *KafkaFetcher
	offsets         map[string]map[string]map[int32]int64 // groupId → topic → partition → next offset
	mu              sync.Mutex
}

// batchMessage is a single Kafka message in the JSON batch sent to lambdas.
type batchMessage struct {
	Topic     string `json:"topic"`
	Partition int32  `json:"partition"`
	Offset    int64  `json:"offset"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	Timestamp string `json:"timestamp"`
}

// newLambdaKafkaConsumer creates a new Kafka consumer for push-based consumption.
func (km *KafkaManager) newLambdaKafkaConsumer(consumerName string, lambdaName string, trigger *common.KafkaTrigger) (*LambdaKafkaConsumer, error) {
	if len(trigger.BootstrapServers) == 0 {
		return nil, fmt.Errorf("no bootstrap servers configured for lambda %s", lambdaName)
	}
	if len(trigger.Topics) == 0 {
		return nil, fmt.Errorf("no topics configured for lambda %s", lambdaName)
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(trigger.BootstrapServers...),
		kgo.ConsumerGroup(trigger.GroupId),
		kgo.ConsumeTopics(trigger.Topics...),
		kgo.SessionTimeout(10 * time.Second),
		kgo.HeartbeatInterval(3 * time.Second),
	}

	if trigger.AutoOffsetReset == "earliest" {
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	} else {
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
	}

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

	manager := &KafkaManager{
		lambdaConsumers: make(map[string]*LambdaKafkaConsumer),
		triggerConfigs:  make(map[string][]common.KafkaTrigger),
		lambdaManager:   lambdaManager,
		cache:           NewMessageCache(int64(cacheSizeMb) * 1024 * 1024),
		fetcher:         NewKafkaFetcher(maxConcurrent),
		offsets:         make(map[string]map[string]map[int32]int64),
	}

	slog.Info("Kafka manager initialized",
		"cache_size_mb", cacheSizeMb,
		"max_concurrent_fetches", maxConcurrent)
	return manager, nil
}

// StartConsuming starts the push-based consuming loop (legacy path).
func (lkc *LambdaKafkaConsumer) StartConsuming() {
	slog.Info("Starting Kafka consumer for lambda",
		"consumer", lkc.consumerName,
		"lambda", lkc.lambdaName,
		"topics", lkc.kafkaTrigger.Topics,
		"brokers", lkc.kafkaTrigger.BootstrapServers,
		"group_id", lkc.kafkaTrigger.GroupId)

	go lkc.consumeLoop()
}

// consumeLoop handles push-based Kafka message consumption (legacy path).
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
					if errors.Is(err.Err, context.DeadlineExceeded) {
						continue
					}
					slog.Warn("Kafka fetch error",
						"lambda", lkc.lambdaName,
						"error", err)
				}
				continue
			}

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

// processMessage handles a single Kafka message by invoking the lambda (legacy push path).
func (lkc *LambdaKafkaConsumer) processMessage(record *kgo.Record) {
	t := common.T0("kafka-message-processing")
	defer t.T1()

	requestPath := fmt.Sprintf("/run/%s/", lkc.lambdaName)
	req, err := http.NewRequest("POST", requestPath, bytes.NewReader(record.Value))
	if err != nil {
		slog.Error("Failed to create request for lambda invocation",
			"lambda", lkc.lambdaName,
			"error", err,
			"topic", record.Topic)
		return
	}
	req.RequestURI = requestPath

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Kafka-Topic", record.Topic)
	req.Header.Set("X-Kafka-Partition", fmt.Sprintf("%d", record.Partition))
	req.Header.Set("X-Kafka-Offset", fmt.Sprintf("%d", record.Offset))
	req.Header.Set("X-Kafka-Group-Id", lkc.kafkaTrigger.GroupId)

	w := httptest.NewRecorder()

	lambdaFunc := lkc.lambdaManager.Get(lkc.lambdaName)
	lambdaFunc.Invoke(w, req)

	slog.Info("Kafka message processed via direct invocation",
		"consumer", lkc.consumerName,
		"lambda", lkc.lambdaName,
		"topic", record.Topic,
		"partition", record.Partition,
		"offset", record.Offset,
		"status", w.Code)
}

func (lkc *LambdaKafkaConsumer) cleanup() {
	slog.Info("Shutting down Kafka consumer for lambda", "lambda", lkc.lambdaName)
	close(lkc.stopChan)
	if lkc.client != nil {
		lkc.client.Close()
	}
	slog.Info("Kafka consumer shutdown complete for lambda", "lambda", lkc.lambdaName)
}

// RegisterLambdaKafkaTriggers stores Kafka trigger configs for a lambda.
// In the pull model, no always-on consumers are started. The configs are used
// when ConsumeFromCache is called to know which brokers/topics to read from.
func (km *KafkaManager) RegisterLambdaKafkaTriggers(lambdaName string, triggers []common.KafkaTrigger) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if len(triggers) == 0 {
		return nil
	}

	// Clean up any existing push-model consumers for this lambda
	for consumerName, consumer := range km.lambdaConsumers {
		if strings.HasPrefix(consumerName, lambdaName+"-") {
			consumer.cleanup()
			delete(km.lambdaConsumers, consumerName)
			slog.Info("Cleaned up existing Kafka consumer for lambda", "lambda", lambdaName, "consumer", consumerName)
		}
	}

	// Store trigger configs with auto-generated group IDs
	stored := make([]common.KafkaTrigger, len(triggers))
	for i, trigger := range triggers {
		trigger.GroupId = fmt.Sprintf("lambda-%s", lambdaName)
		stored[i] = trigger
	}
	km.triggerConfigs[lambdaName] = stored

	slog.Info("Registered Kafka trigger configs for lambda",
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

// ConsumeFromCache reads messages from the shared cache for a lambda's registered
// Kafka triggers. On cache miss, it fetches from Kafka via the consumer pool, caches
// the results, then invokes the lambda with a batched JSON payload.
//
// Returns the number of messages consumed and any error.
func (km *KafkaManager) ConsumeFromCache(lambdaName string, w http.ResponseWriter) (int, error) {
	t := common.T0("kafka-consume-from-cache")
	defer t.T1()

	km.mu.Lock()
	triggers, ok := km.triggerConfigs[lambdaName]
	km.mu.Unlock()

	if !ok || len(triggers) == 0 {
		return 0, fmt.Errorf("no Kafka triggers registered for lambda %s", lambdaName)
	}

	batchSize := common.Conf.Kafka_batch_size
	if batchSize <= 0 {
		batchSize = 100
	}

	var allMessages []batchMessage

	for _, trigger := range triggers {
		groupId := trigger.GroupId

		for _, topic := range trigger.Topics {
			// For pull model, we consume partition 0 by default.
			// TODO: support multi-partition consumption by discovering partition count.
			partition := int32(0)

			km.mu.Lock()
			startOffset := km.getOffset(groupId, topic, partition)
			km.mu.Unlock()

			// Try cache first
			cached, missOffset := km.cache.GetBatch(topic, partition, startOffset, batchSize)

			if missOffset != -1 {
				// Cache miss — fetch from Kafka
				slog.Info("Cache miss, fetching from Kafka",
					"lambda", lambdaName,
					"topic", topic,
					"partition", partition,
					"missOffset", missOffset,
					"batchSize", batchSize)

				fetchCtx, fetchCancel := context.WithTimeout(context.Background(), 15*time.Second)
				records, err := km.fetcher.Fetch(
					fetchCtx, trigger.BootstrapServers, topic, partition, missOffset, batchSize,
				)
				fetchCancel()
				if err != nil {
					slog.Error("Failed to fetch from Kafka",
						"lambda", lambdaName,
						"topic", topic,
						"error", err)
					continue
				}

				// Cache fetched records
				for _, r := range records {
					headers := make(map[string]string)
					for _, h := range r.Headers {
						headers[h.Key] = string(h.Value)
					}
					msg := &CachedMessage{
						Key:       r.Key,
						Value:     r.Value,
						Headers:   headers,
						Timestamp: r.Timestamp,
						size:      int64(len(r.Key) + len(r.Value) + 64), // approximate overhead
					}
					km.cache.Put(CacheKey{
						Topic:     r.Topic,
						Partition: r.Partition,
						Offset:    r.Offset,
					}, msg)
				}

				// Re-read from cache to get the full contiguous batch
				cached, _ = km.cache.GetBatch(topic, partition, startOffset, batchSize)
			}

			// Build batch messages
			for i, msg := range cached {
				allMessages = append(allMessages, batchMessage{
					Topic:     topic,
					Partition: partition,
					Offset:    startOffset + int64(i),
					Key:       base64.StdEncoding.EncodeToString(msg.Key),
					Value:     base64.StdEncoding.EncodeToString(msg.Value),
					Timestamp: msg.Timestamp.Format(time.RFC3339),
				})
			}

			// Advance offset past consumed messages
			if len(cached) > 0 {
				newOffset := startOffset + int64(len(cached))
				km.mu.Lock()
				km.setOffset(groupId, topic, partition, newOffset)
				km.mu.Unlock()

				slog.Info("Advanced consumer offset",
					"lambda", lambdaName,
					"topic", topic,
					"partition", partition,
					"oldOffset", startOffset,
					"newOffset", newOffset)
			}
		}
	}

	if len(allMessages) == 0 {
		slog.Info("No messages available for lambda", "lambda", lambdaName)
		return 0, nil
	}

	// Serialize batch and invoke lambda
	batchJSON, err := json.Marshal(allMessages)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal message batch: %w", err)
	}

	requestPath := fmt.Sprintf("/run/%s/", lambdaName)
	req, err := http.NewRequest("POST", requestPath, bytes.NewReader(batchJSON))
	if err != nil {
		return 0, fmt.Errorf("failed to create synthetic request: %w", err)
	}
	req.RequestURI = requestPath

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Kafka-Batch", "true")
	req.Header.Set("X-Kafka-Message-Count", fmt.Sprintf("%d", len(allMessages)))
	req.Header.Set("X-Kafka-Group-Id", triggers[0].GroupId)

	lambdaFunc := km.lambdaManager.Get(lambdaName)
	lambdaFunc.Invoke(w, req)

	slog.Info("Kafka batch consumed from cache and lambda invoked",
		"lambda", lambdaName,
		"messageCount", len(allMessages),
		"status", "invoked")

	return len(allMessages), nil
}

// UnregisterLambdaKafkaTriggers removes Kafka triggers for a lambda function.
func (km *KafkaManager) UnregisterLambdaKafkaTriggers(lambdaName string) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Clean up push-model consumers
	for consumerName, consumer := range km.lambdaConsumers {
		if strings.HasPrefix(consumerName, lambdaName+"-") {
			consumer.cleanup()
			delete(km.lambdaConsumers, consumerName)
			slog.Info("Unregistered Kafka consumer for lambda", "lambda", lambdaName)
		}
	}

	// Remove trigger configs
	delete(km.triggerConfigs, lambdaName)

	slog.Info("Unregistered Kafka triggers for lambda", "lambda", lambdaName)
}

// cleanup shuts down all consumers, the consumer pool, and clears the cache.
func (km *KafkaManager) cleanup() {
	slog.Info("Shutting down Kafka manager")

	km.mu.Lock()
	defer km.mu.Unlock()

	for lambdaName, consumer := range km.lambdaConsumers {
		consumer.cleanup()
		slog.Info("Cleaned up Kafka consumer for lambda", "lambda", lambdaName)
	}
	km.lambdaConsumers = make(map[string]*LambdaKafkaConsumer)
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

// HandleKafkaConsume handles pull-based Kafka consumption for a lambda.
// POST /kafka/consume/<lambda> reads messages from the cache (fetching on miss)
// and invokes the lambda with a batched JSON payload.
func HandleKafkaConsume(kafkaManager *KafkaManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		lambdaName := strings.TrimPrefix(r.URL.Path, "/kafka/consume/")
		if lambdaName == "" {
			http.Error(w, "lambda name required", http.StatusBadRequest)
			return
		}

		// Use a recorder to capture the lambda's response so we can
		// wrap it with cache metadata for the caller.
		recorder := httptest.NewRecorder()

		count, err := kafkaManager.ConsumeFromCache(lambdaName, recorder)
		if err != nil {
			slog.Error("Failed to consume from cache",
				"lambda", lambdaName,
				"error", err)
			http.Error(w, fmt.Sprintf("consume error: %v", err), http.StatusInternalServerError)
			return
		}

		type Response struct {
			Status       string `json:"status"`
			Lambda       string `json:"lambda"`
			MessageCount int    `json:"message_count"`
			LambdaStatus int    `json:"lambda_status"`
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Status:       "success",
			Lambda:       lambdaName,
			MessageCount: count,
			LambdaStatus: recorder.Code,
		})
	}
}

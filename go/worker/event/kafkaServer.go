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
	"strconv"
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
	SetOffset(topic string, partition int32, offset int64)
	Close()
}

// kgoClientWrapper wraps *kgo.Client to implement KafkaClient, adding the
// SetOffset method that maps to kgo's SetOffsets API.
type kgoClientWrapper struct {
	client *kgo.Client
}

func (w *kgoClientWrapper) PollFetches(ctx context.Context) kgo.Fetches {
	return w.client.PollFetches(ctx)
}

func (w *kgoClientWrapper) SetOffset(topic string, partition int32, offset int64) {
	w.client.SetOffsets(map[string]map[int32]kgo.EpochOffset{
		topic: {partition: {Offset: offset}},
	})
}

func (w *kgoClientWrapper) Close() {
	w.client.Close()
}

// cacheKey uniquely identifies a Kafka record by its topic, partition, and offset.
type cacheKey struct {
	topic     string
	partition int32
	offset    int64
}

// seekState tracks the current seek position when replaying from cache.
type seekState struct {
	topic     string
	partition int32
	offset    int64 // next offset to serve from cache
}

// seekRequest is returned by processMessage when the lambda requests a seek.
type seekRequest struct {
	offset int64
}

const defaultCacheSize = 1024

// cachedKafkaClient wraps a KafkaClient and caches records in an LRU map keyed
// by {topic, partition, offset}. When a seek is active, PollFetches serves
// records from the cache. On cache miss, the seek ends and normal polling resumes.
type cachedKafkaClient struct {
	underlying KafkaClient
	cache      map[cacheKey]*kgo.Record
	evictOrder []cacheKey // front = least recently used
	maxSize    int
	seekTarget *seekState
}

func newCachedKafkaClient(underlying KafkaClient, maxSize int) *cachedKafkaClient {
	return &cachedKafkaClient{
		underlying: underlying,
		cache:      make(map[cacheKey]*kgo.Record),
		maxSize:    maxSize,
	}
}

// Seek sets the seek target so that subsequent PollFetches calls serve from cache.
func (c *cachedKafkaClient) Seek(topic string, partition int32, offset int64) {
	c.seekTarget = &seekState{topic: topic, partition: partition, offset: offset}
}

// PollFetches serves from cache when seeking, otherwise delegates to the underlying client.
func (c *cachedKafkaClient) PollFetches(ctx context.Context) kgo.Fetches {
	if c.seekTarget != nil {
		key := cacheKey{
			topic:     c.seekTarget.topic,
			partition: c.seekTarget.partition,
			offset:    c.seekTarget.offset,
		}
		record, ok := c.cache[key]
		if ok {
			c.touchLRU(key)
			c.seekTarget.offset++
			return makeSingleRecordFetches(record)
		}
		// Cache miss — seek on Kafka to fetch the record from the broker
		slog.Info("Seek cache miss, fetching from Kafka",
			"topic", c.seekTarget.topic,
			"partition", c.seekTarget.partition,
			"offset", c.seekTarget.offset)
		c.underlying.SetOffset(c.seekTarget.topic, c.seekTarget.partition, c.seekTarget.offset)
		fetches := c.underlying.PollFetches(ctx)
		fetches.EachRecord(func(record *kgo.Record) {
			c.put(cacheKey{topic: record.Topic, partition: record.Partition, offset: record.Offset}, record)
		})

		// Check cache again after fetching
		record, ok = c.cache[key]
		if ok {
			c.touchLRU(key)
			c.seekTarget.offset++
			return makeSingleRecordFetches(record)
		}
		// Still not found (offset may be past the end of the partition) — give up
		slog.Warn("Seek offset not available from Kafka, resuming normal polling",
			"topic", c.seekTarget.topic,
			"partition", c.seekTarget.partition,
			"offset", c.seekTarget.offset)
		c.seekTarget = nil
		return fetches
	}

	fetches := c.underlying.PollFetches(ctx)
	fetches.EachRecord(func(record *kgo.Record) {
		c.put(cacheKey{topic: record.Topic, partition: record.Partition, offset: record.Offset}, record)
	})
	return fetches
}

func (c *cachedKafkaClient) SetOffset(topic string, partition int32, offset int64) {
	c.underlying.SetOffset(topic, partition, offset)
}

func (c *cachedKafkaClient) Close() {
	c.underlying.Close()
}

// put adds a record to the cache, evicting the LRU entry if at capacity.
func (c *cachedKafkaClient) put(key cacheKey, record *kgo.Record) {
	if _, exists := c.cache[key]; exists {
		c.touchLRU(key)
		return
	}
	if len(c.cache) >= c.maxSize {
		evictKey := c.evictOrder[0]
		c.evictOrder = c.evictOrder[1:]
		delete(c.cache, evictKey)
	}
	c.cache[key] = record
	c.evictOrder = append(c.evictOrder, key)
}

// touchLRU moves a key to the back of the eviction order (most recently used).
func (c *cachedKafkaClient) touchLRU(key cacheKey) {
	for i, k := range c.evictOrder {
		if k == key {
			c.evictOrder = append(c.evictOrder[:i], c.evictOrder[i+1:]...)
			c.evictOrder = append(c.evictOrder, key)
			return
		}
	}
}

// makeSingleRecordFetches wraps a single record into the kgo.Fetches structure.
func makeSingleRecordFetches(record *kgo.Record) kgo.Fetches {
	return kgo.Fetches{{
		Topics: []kgo.FetchTopic{{
			Topic: record.Topic,
			Partitions: []kgo.FetchPartition{{
				Partition: record.Partition,
				Records:   []*kgo.Record{record},
			}},
		}},
	}}
}

// LambdaInvoker abstracts the lambda invocation layer for testability
type LambdaInvoker interface {
	Invoke(lambdaName string, w http.ResponseWriter, r *http.Request)
}

// lambdaMgrInvoker wraps *lambda.LambdaMgr to implement LambdaInvoker
type lambdaMgrInvoker struct {
	mgr *lambda.LambdaMgr
}

func (i *lambdaMgrInvoker) Invoke(lambdaName string, w http.ResponseWriter, r *http.Request) {
	f := i.mgr.Get(lambdaName)
	f.Invoke(w, r)
}

// LambdaKafkaConsumer manages Kafka consumption for a specific lambda function
type LambdaKafkaConsumer struct {
	consumerName string // Unique name for this consumer
	lambdaName   string // lambda function name
	kafkaTrigger *common.KafkaTrigger
	client       KafkaClient          // used for PollFetches/Close (may be a cachedKafkaClient)
	cache        *cachedKafkaClient   // typed reference for Seek; same object as client when caching is enabled
	invoker      LambdaInvoker        // Abstraction for lambda invocation
	stopChan     chan struct{}         // Shutdown signal for this consumer
	// When this channel is closed, the goroutine for the consumer exits
	errorCount   int                  // Number of non-timeout Kafka client errors encountered
}

// KafkaManager manages multiple lambda-specific Kafka consumers
type KafkaManager struct {
	lambdaConsumers map[string]*LambdaKafkaConsumer // lambdaName -> consumer
	invoker         LambdaInvoker                   // Abstraction for lambda invocation
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

	cached := newCachedKafkaClient(&kgoClientWrapper{client: client}, defaultCacheSize)
	return &LambdaKafkaConsumer{
		consumerName: consumerName,
		lambdaName:   lambdaName,
		kafkaTrigger: trigger,
		client:       cached,
		cache:        cached,
		invoker:      km.invoker,
		stopChan:     make(chan struct{}),
	}, nil
}

// NewKafkaManager creates and configures a new Kafka manager
func NewKafkaManager(lambdaManager *lambda.LambdaMgr) (*KafkaManager, error) {
	manager := &KafkaManager{
		lambdaConsumers: make(map[string]*LambdaKafkaConsumer),
		invoker:         &lambdaMgrInvoker{mgr: lambdaManager},
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

					lkc.errorCount++
					// TODO: Surface Kafka consumer errors to lambda developers by invoking an error
					// handler lambda function. Could allow lambdas to specify an onError callback in
					// ol.yaml that gets invoked with error details.
					slog.Warn("Kafka fetch error",
						"lambda", lkc.lambdaName,
						"error", err)
				}
				continue
			}

			// Process each record. Manual iteration (instead of EachRecord) lets
			// us break out mid-batch when a seek is requested.
			for _, record := range fetches.Records() {
				slog.Info("Received Kafka message for lambda",
					"consumer", lkc.consumerName,
					"lambda", lkc.lambdaName,
					"topic", record.Topic,
					"partition", record.Partition,
					"offset", record.Offset,
					"size", len(record.Value))
				if seek := lkc.processMessage(record); seek != nil && lkc.cache != nil {
					lkc.cache.Seek(record.Topic, record.Partition, seek.offset)
					break // next PollFetches will serve from cache
				}
			}
		}
	}
}

// processMessage handles a single Kafka message by invoking the lambda function directly.
// If the lambda returns an X-Kafka-Seek-Offset header, the corresponding seekRequest is returned.
func (lkc *LambdaKafkaConsumer) processMessage(record *kgo.Record) *seekRequest {
	t := common.T0("kafka-message-processing")
	defer t.T1()

	// Create synthetic HTTP request from Kafka message
	// Path must be /run/<lambda-name>/ for the Python runtime to parse correctly
	requestPath := fmt.Sprintf("/run/%s/", lkc.lambdaName)
	req, err := http.NewRequest("POST", requestPath, bytes.NewReader(record.Value))
	if err != nil {
		slog.Error("Failed to create request for lambda invocation",
			"lambda", lkc.lambdaName,
			"error", err,
			"topic", record.Topic)
		return nil
	}
	// RequestURI must be set explicitly for synthetic requests (http.NewRequest doesn't set it)
	req.RequestURI = requestPath

	// Set headers with Kafka metadata (The X- prefix indicates a custom non-standard header)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Kafka-Topic", record.Topic)
	req.Header.Set("X-Kafka-Partition", fmt.Sprintf("%d", record.Partition))
	req.Header.Set("X-Kafka-Offset", fmt.Sprintf("%d", record.Offset))
	req.Header.Set("X-Kafka-Group-Id", lkc.kafkaTrigger.GroupId)

	// Create response recorder to capture lambda output.
	w := httptest.NewRecorder()

	// Invoke the lambda function directly
	lkc.invoker.Invoke(lkc.lambdaName, w, req)

	// Log the result
	slog.Info("Kafka message processed via direct invocation",
		"consumer", lkc.consumerName,
		"lambda", lkc.lambdaName,
		"topic", record.Topic,
		"partition", record.Partition,
		"offset", record.Offset,
		"status", w.Code)

	// Check if the lambda requested a seek via response header
	if seekStr := w.Header().Get("X-Kafka-Seek-Offset"); seekStr != "" {
		seekOffset, err := strconv.ParseInt(seekStr, 10, 64)
		if err != nil {
			slog.Warn("Invalid X-Kafka-Seek-Offset header",
				"lambda", lkc.lambdaName,
				"value", seekStr,
				"error", err)
			return nil
		}
		slog.Info("Lambda requested seek",
			"lambda", lkc.lambdaName,
			"topic", record.Topic,
			"partition", record.Partition,
			"current_offset", record.Offset,
			"seek_offset", seekOffset)
		return &seekRequest{offset: seekOffset}
	}
	return nil
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

package event

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// OnDemandConsumer wraps a kgo.Client created on-demand for a specific broker+topic combination.
type OnDemandConsumer struct {
	client   *kgo.Client
	brokers  []string
	topic    string
	lastUsed time.Time
	mu       sync.Mutex
}

// ConsumerPool manages a pool of on-demand Kafka consumers keyed by broker+topic.
// Consumers are created lazily on cache miss and cleaned up after an idle timeout.
type ConsumerPool struct {
	consumers   map[string]*OnDemandConsumer // key: "broker1,broker2|topic"
	mu          sync.Mutex
	idleTimeout time.Duration
	stopChan    chan struct{}
}

// consumerKey builds a map key from brokers and topic.
func consumerKey(brokers []string, topic string) string {
	sorted := make([]string, len(brokers))
	copy(sorted, brokers)
	sort.Strings(sorted)
	return strings.Join(sorted, ",") + "|" + topic
}

// NewConsumerPool creates a ConsumerPool that cleans up idle consumers.
func NewConsumerPool(idleTimeout time.Duration) *ConsumerPool {
	cp := &ConsumerPool{
		consumers:   make(map[string]*OnDemandConsumer),
		idleTimeout: idleTimeout,
		stopChan:    make(chan struct{}),
	}
	go cp.cleanupLoop()
	return cp
}

// Fetch retrieves a batch of records starting at startOffset for the given
// topic and partition. It reuses an existing consumer or creates a new one.
func (cp *ConsumerPool) Fetch(brokers []string, topic string, partition int32, startOffset int64, count int) ([]*kgo.Record, error) {
	consumer, err := cp.getOrCreate(brokers, topic, partition, startOffset)
	if err != nil {
		return nil, err
	}

	consumer.mu.Lock()
	defer consumer.mu.Unlock()
	consumer.lastUsed = time.Now()

	var records []*kgo.Record
	deadline := time.After(10 * time.Second)

	for len(records) < count {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		fetches := consumer.client.PollFetches(ctx)
		cancel()

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, fe := range errs {
				if fe.Err == context.DeadlineExceeded {
					continue
				}
				slog.Warn("ConsumerPool fetch error",
					"topic", topic,
					"partition", partition,
					"error", fe.Err)
			}
		}

		fetches.EachRecord(func(r *kgo.Record) {
			if r.Partition == partition && r.Offset >= startOffset {
				records = append(records, r)
			}
		})

		if len(records) >= count {
			break
		}

		select {
		case <-deadline:
			slog.Info("ConsumerPool fetch deadline reached",
				"topic", topic,
				"partition", partition,
				"fetched", len(records),
				"requested", count)
			return records, nil
		default:
		}
	}

	return records, nil
}

// getOrCreate returns an existing consumer for the broker+topic or creates one
// configured to consume the specific partition starting at the given offset.
func (cp *ConsumerPool) getOrCreate(brokers []string, topic string, partition int32, startOffset int64) (*OnDemandConsumer, error) {
	key := consumerKey(brokers, topic)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	// For on-demand fetches we always create a fresh consumer targeting the
	// exact partition+offset so we don't collide with other concurrent fetches.
	// Close the old consumer for this key if it exists.
	if old, ok := cp.consumers[key]; ok {
		old.client.Close()
		delete(cp.consumers, key)
	}

	offsets := make(map[string]map[int32]kgo.Offset)
	offsets[topic] = map[int32]kgo.Offset{
		partition: kgo.NewOffset().At(startOffset),
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ConsumePartitions(offsets),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create on-demand consumer for %s/%s: %w", topic, brokers, err)
	}

	consumer := &OnDemandConsumer{
		client:   client,
		brokers:  brokers,
		topic:    topic,
		lastUsed: time.Now(),
	}

	cp.consumers[key] = consumer
	slog.Info("Created on-demand consumer",
		"topic", topic,
		"partition", partition,
		"startOffset", startOffset)

	return consumer, nil
}

// cleanupLoop periodically closes consumers that have been idle beyond idleTimeout.
func (cp *ConsumerPool) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cp.stopChan:
			return
		case <-ticker.C:
			cp.mu.Lock()
			now := time.Now()
			for key, consumer := range cp.consumers {
				consumer.mu.Lock()
				idle := now.Sub(consumer.lastUsed)
				consumer.mu.Unlock()

				if idle > cp.idleTimeout {
					slog.Info("Closing idle consumer",
						"topic", consumer.topic,
						"idle", idle)
					consumer.client.Close()
					delete(cp.consumers, key)
				}
			}
			cp.mu.Unlock()
		}
	}
}

// Close shuts down all consumers and stops the cleanup loop.
func (cp *ConsumerPool) Close() {
	close(cp.stopChan)

	cp.mu.Lock()
	defer cp.mu.Unlock()

	for key, consumer := range cp.consumers {
		consumer.client.Close()
		delete(cp.consumers, key)
	}

	slog.Info("ConsumerPool closed")
}

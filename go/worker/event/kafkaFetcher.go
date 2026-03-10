package event

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// KafkaFetcher retrieves Kafka messages, using a shared MessageCache as a
// read-through layer. On cache hit the message is returned immediately.
// On miss a short-lived consumer is created (bounded by a semaphore),
// the result is cached, and then returned.
type KafkaFetcher struct {
	cache *MessageCache
	sem   chan struct{}
}

// NewKafkaFetcher creates a KafkaFetcher backed by the given cache.
func NewKafkaFetcher(cache *MessageCache, maxConcurrent int) *KafkaFetcher {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}
	return &KafkaFetcher{
		cache: cache,
		sem:   make(chan struct{}, maxConcurrent),
	}
}

// Get returns the message at the given topic/partition/offset.
// It checks the cache first; on miss it fetches from Kafka, caches
// the result, and returns it. Returns nil if no message is available.
func (kf *KafkaFetcher) Get(ctx context.Context, brokers []string, topic string, partition int32, offset int64) (*CachedMessage, error) {
	key := CacheKey{Topic: topic, Partition: partition, Offset: offset}

	if msg, hit := kf.cache.Get(key); hit {
		return msg, nil
	}

	// Cache miss — fetch from Kafka
	record, err := kf.fetchFromKafka(ctx, brokers, topic, partition, offset)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}

	headers := make(map[string]string)
	for _, h := range record.Headers {
		headers[h.Key] = string(h.Value)
	}
	msg := &CachedMessage{
		Key:       record.Key,
		Value:     record.Value,
		Headers:   headers,
		Timestamp: record.Timestamp,
		size:      int64(len(record.Key) + len(record.Value) + 64),
	}
	kf.cache.Put(CacheKey{
		Topic:     record.Topic,
		Partition: record.Partition,
		Offset:    record.Offset,
	}, msg)

	return msg, nil
}

// fetchFromKafka creates a short-lived consumer, reads a single record,
// and closes the consumer. Blocks if the semaphore is full.
func (kf *KafkaFetcher) fetchFromKafka(ctx context.Context, brokers []string, topic string, partition int32, offset int64) (*kgo.Record, error) {
	select {
	case kf.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-kf.sem }()

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumePartitions(map[string]map[int32]kgo.Offset{
			topic: {
				partition: kgo.NewOffset().At(offset),
			},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer for %s partition %d: %w", topic, partition, err)
	}
	defer client.Close()

	deadline := time.After(10 * time.Second)

	for {
		pollCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		fetches := client.PollFetches(pollCtx)
		cancel()

		if errs := fetches.Errors(); len(errs) > 0 {
			for _, fe := range errs {
				if fe.Err == context.DeadlineExceeded {
					continue
				}
				slog.Warn("KafkaFetcher fetch error",
					"topic", topic,
					"partition", partition,
					"error", fe.Err)
			}
		}

		var found *kgo.Record
		fetches.EachRecord(func(r *kgo.Record) {
			if found == nil && r.Partition == partition && r.Offset >= offset {
				found = r
			}
		})

		if found != nil {
			return found, nil
		}

		select {
		case <-deadline:
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
}

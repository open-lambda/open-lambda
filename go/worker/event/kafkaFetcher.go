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
// On miss it prefetches multiple messages from Kafka, caches them all,
// and returns the requested one.
type KafkaFetcher struct {
	cache         *MessageCache
	sem           chan struct{}
	prefetchCount int
}

// NewKafkaFetcher creates a KafkaFetcher backed by the given cache.
func NewKafkaFetcher(cache *MessageCache, maxConcurrent int, prefetchCount int) *KafkaFetcher {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}
	if prefetchCount <= 0 {
		prefetchCount = 5
	}
	return &KafkaFetcher{
		cache:         cache,
		sem:           make(chan struct{}, maxConcurrent),
		prefetchCount: prefetchCount,
	}
}

// Get returns the message at the given topic/partition/offset.
// It checks the cache first; on miss it fetches prefetchCount messages
// starting at offset, caches them all, and returns the requested one.
// Returns nil if no message is available.
func (kf *KafkaFetcher) Get(ctx context.Context, brokers []string, topic string, partition int32, offset int64) (*CachedMessage, error) {
	key := CacheKey{Topic: topic, Partition: partition, Offset: offset}

	if msg, hit := kf.cache.Get(key); hit {
		return msg, nil
	}

	// Cache miss — prefetch from Kafka
	records, err := kf.fetchFromKafka(ctx, brokers, topic, partition, offset, kf.prefetchCount)
	if err != nil {
		return nil, err
	}

	// Cache all fetched records
	var result *CachedMessage
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
			size:      int64(len(r.Key) + len(r.Value) + 64),
		}
		kf.cache.Put(CacheKey{
			Topic:     r.Topic,
			Partition: r.Partition,
			Offset:    r.Offset,
		}, msg)

		if r.Offset == offset {
			result = msg
		}
	}

	return result, nil
}

// fetchFromKafka creates a short-lived consumer, reads up to count records
// starting at offset, and closes the consumer. Blocks if the semaphore is full.
func (kf *KafkaFetcher) fetchFromKafka(ctx context.Context, brokers []string, topic string, partition int32, offset int64, count int) ([]*kgo.Record, error) {
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

	var records []*kgo.Record
	deadline := time.After(10 * time.Second)

	for len(records) < count {
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

		fetches.EachRecord(func(r *kgo.Record) {
			if r.Partition == partition && r.Offset >= offset {
				records = append(records, r)
			}
		})

		if len(records) >= count {
			break
		}

		select {
		case <-deadline:
			return records, nil
		case <-ctx.Done():
			return records, ctx.Err()
		default:
		}
	}

	return records, nil
}

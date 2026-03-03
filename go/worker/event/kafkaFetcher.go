package event

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// KafkaFetcher creates short-lived Kafka consumers to fetch batches of records.
// A semaphore limits the number of concurrent consumers to avoid resource exhaustion;
// callers block until a slot is available.
type KafkaFetcher struct {
	sem chan struct{}
}

// NewKafkaFetcher creates a KafkaFetcher that allows at most maxConcurrent
// simultaneous Kafka consumers.
func NewKafkaFetcher(maxConcurrent int) *KafkaFetcher {
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}
	return &KafkaFetcher{
		sem: make(chan struct{}, maxConcurrent),
	}
}

// Fetch creates a short-lived consumer for the given broker+topic+partition,
// reads up to count records starting at startOffset, then closes the consumer.
// Blocks if the concurrency limit has been reached.
func (kf *KafkaFetcher) Fetch(ctx context.Context, brokers []string, topic string, partition int32, startOffset int64, count int) ([]*kgo.Record, error) {
	// Acquire semaphore slot (blocks if at capacity)
	select {
	case kf.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-kf.sem }()

	offsets := map[string]map[int32]kgo.Offset{
		topic: {
			partition: kgo.NewOffset().At(startOffset),
		},
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumePartitions(offsets),
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
			if r.Partition == partition && r.Offset >= startOffset {
				records = append(records, r)
			}
		})

		if len(records) >= count {
			break
		}

		select {
		case <-deadline:
			slog.Info("KafkaFetcher deadline reached",
				"topic", topic,
				"partition", partition,
				"fetched", len(records),
				"requested", count)
			return records, nil
		case <-ctx.Done():
			return records, ctx.Err()
		default:
		}
	}

	return records, nil
}

package event

import (
	"context"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"
)

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

const defaultCacheSizeMB = 16

// cachedKafkaClient wraps a KafkaClient and caches records in an LRU map keyed
// by {topic, partition, offset}. When a seek is active, PollFetches serves
// records from the cache. On cache miss, it calls Seek on the underlying
// client so the next poll fetches from the right position.
type cachedKafkaClient struct {
	underlying   KafkaClient
	cache        map[cacheKey]*kgo.Record
	evictOrder   []cacheKey // front = least recently used
	maxSizeBytes int64
	currentSize  int64
	seekTarget   *seekState
}

func newCachedKafkaClient(underlying KafkaClient, maxSizeMB int) *cachedKafkaClient {
	return &cachedKafkaClient{
		underlying:   underlying,
		cache:        make(map[cacheKey]*kgo.Record),
		maxSizeBytes: int64(maxSizeMB) * 1024 * 1024,
	}
}

// recordSize returns the approximate in-memory size of a cached record in bytes.
func recordSize(r *kgo.Record) int64 {
	size := int64(len(r.Key) + len(r.Value) + len(r.Topic))
	for _, h := range r.Headers {
		size += int64(len(h.Key) + len(h.Value))
	}
	return size
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
		// Cache miss — tell the underlying client to fetch from this offset.
		// The next normal PollFetches will get records starting here.
		slog.Info("Seek cache miss, setting offset on underlying client",
			"topic", c.seekTarget.topic,
			"partition", c.seekTarget.partition,
			"offset", c.seekTarget.offset)
		c.underlying.Seek(c.seekTarget.topic, c.seekTarget.partition, c.seekTarget.offset)
		c.seekTarget = nil
	}

	fetches := c.underlying.PollFetches(ctx)
	fetches.EachRecord(func(record *kgo.Record) {
		c.put(cacheKey{topic: record.Topic, partition: record.Partition, offset: record.Offset}, record)
	})
	return fetches
}

func (c *cachedKafkaClient) Close() {
	c.underlying.Close()
}

// put adds a record to the cache, evicting LRU entries until there is room.
func (c *cachedKafkaClient) put(key cacheKey, record *kgo.Record) {
	if _, exists := c.cache[key]; exists {
		c.touchLRU(key)
		return
	}
	newSize := recordSize(record)
	for c.currentSize+newSize > c.maxSizeBytes && len(c.evictOrder) > 0 {
		evictKey := c.evictOrder[0]
		c.evictOrder = c.evictOrder[1:]
		c.currentSize -= recordSize(c.cache[evictKey])
		delete(c.cache, evictKey)
	}
	c.cache[key] = record
	c.evictOrder = append(c.evictOrder, key)
	c.currentSize += newSize
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

package event

import (
	"context"
	"testing"
	"time"
)

func newTestFetcher(maxConcurrent int) *KafkaFetcher {
	cache := NewMessageCache(1024 * 1024)
	return NewKafkaFetcher(cache, maxConcurrent, 5)
}

func TestNewKafkaFetcher_DefaultConcurrency(t *testing.T) {
	kf := newTestFetcher(0)
	if cap(kf.sem) != 10 {
		t.Fatalf("expected default capacity 10, got %d", cap(kf.sem))
	}
}

func TestNewKafkaFetcher_CustomConcurrency(t *testing.T) {
	kf := newTestFetcher(5)
	if cap(kf.sem) != 5 {
		t.Fatalf("expected capacity 5, got %d", cap(kf.sem))
	}
}

func TestKafkaFetcher_CacheHit(t *testing.T) {
	cache := NewMessageCache(1024 * 1024)
	kf := NewKafkaFetcher(cache, 1, 5)

	// Pre-populate the cache
	key := CacheKey{Topic: "t", Partition: 0, Offset: 42}
	cache.Put(key, &CachedMessage{
		Value: []byte("cached-value"),
		size:  100,
	})

	// Get should return the cached message without hitting Kafka
	msg, err := kf.Get(context.Background(), []string{"localhost:9092"}, "t", 0, 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected cache hit, got nil")
	}
	if string(msg.Value) != "cached-value" {
		t.Fatalf("expected 'cached-value', got %q", string(msg.Value))
	}
}

func TestKafkaFetcher_SemaphoreBlocksAtCapacity(t *testing.T) {
	kf := newTestFetcher(1)

	// Fill the semaphore slot
	kf.sem <- struct{}{}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Get should block on semaphore and then fail with context deadline
	_, err := kf.Get(ctx, []string{"localhost:9092"}, "test-topic", 0, 0)
	if err == nil {
		t.Fatal("expected error when semaphore is full and context expires")
	}
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}

	// Release the slot
	<-kf.sem
}

func TestKafkaFetcher_SemaphoreReleasedAfterFetch(t *testing.T) {
	kf := newTestFetcher(1)

	// Get will fail (no real broker) but should still release the semaphore slot
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	kf.Get(ctx, []string{"localhost:19092"}, "nonexistent", 0, 0)

	// Verify the semaphore slot was released by acquiring it without blocking
	select {
	case kf.sem <- struct{}{}:
		<-kf.sem
	default:
		t.Fatal("semaphore slot was not released after Get")
	}
}

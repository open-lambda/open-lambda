package event

import (
	"context"
	"testing"
	"time"
)

func TestNewKafkaFetcher_DefaultConcurrency(t *testing.T) {
	kf := NewKafkaFetcher(0)
	if cap(kf.sem) != 10 {
		t.Fatalf("expected default capacity 10, got %d", cap(kf.sem))
	}
}

func TestNewKafkaFetcher_CustomConcurrency(t *testing.T) {
	kf := NewKafkaFetcher(5)
	if cap(kf.sem) != 5 {
		t.Fatalf("expected capacity 5, got %d", cap(kf.sem))
	}
}

func TestKafkaFetcher_SemaphoreBlocksAtCapacity(t *testing.T) {
	kf := NewKafkaFetcher(1)

	// Fill the semaphore slot
	kf.sem <- struct{}{}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Fetch should block and then fail with context deadline since the slot is taken
	_, err := kf.Fetch(ctx, []string{"localhost:9092"}, "test-topic", 0, 0, 1)
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
	kf := NewKafkaFetcher(1)

	// Fetch will fail (no real broker) but should still release the semaphore slot
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This will error because there's no broker, but the semaphore should be released
	kf.Fetch(ctx, []string{"localhost:19092"}, "nonexistent", 0, 0, 1)

	// Verify the semaphore slot was released by acquiring it without blocking
	select {
	case kf.sem <- struct{}{}:
		<-kf.sem // clean up
	default:
		t.Fatal("semaphore slot was not released after Fetch")
	}
}

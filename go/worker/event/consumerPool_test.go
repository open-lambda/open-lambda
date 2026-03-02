package event

import (
	"sort"
	"testing"
	"time"
)

func TestConsumerKey(t *testing.T) {
	// Order of brokers should not matter
	key1 := consumerKey([]string{"b1:9092", "b2:9092"}, "topic1")
	key2 := consumerKey([]string{"b2:9092", "b1:9092"}, "topic1")

	if key1 != key2 {
		t.Fatalf("expected same key regardless of broker order: %q vs %q", key1, key2)
	}

	// Different topics should produce different keys
	key3 := consumerKey([]string{"b1:9092"}, "topic2")
	if key1 == key3 {
		t.Fatal("expected different keys for different topics")
	}
}

func TestConsumerKey_SortedBrokers(t *testing.T) {
	brokers := []string{"c:9092", "a:9092", "b:9092"}
	key := consumerKey(brokers, "t")

	sorted := make([]string, len(brokers))
	copy(sorted, brokers)
	sort.Strings(sorted)

	// Original slice should not be mutated
	if brokers[0] != "c:9092" {
		t.Fatal("consumerKey should not mutate the input slice")
	}

	expected := "a:9092,b:9092,c:9092|t"
	if key != expected {
		t.Fatalf("expected %q, got %q", expected, key)
	}
}

func TestNewConsumerPool(t *testing.T) {
	cp := NewConsumerPool(30 * time.Second)
	defer cp.Close()

	if len(cp.consumers) != 0 {
		t.Fatal("expected empty consumer map")
	}
}

func TestConsumerPool_Close(t *testing.T) {
	cp := NewConsumerPool(30 * time.Second)
	cp.Close()

	// Closing again should not panic (stopChan already closed)
	// The cleanup loop should have exited
	cp.mu.Lock()
	if len(cp.consumers) != 0 {
		t.Fatal("expected empty consumer map after close")
	}
	cp.mu.Unlock()
}

package event

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func makeMsg(value string, size int64) *CachedMessage {
	return &CachedMessage{
		Value:     []byte(value),
		Timestamp: time.Now(),
		size:      size,
	}
}

func TestMessageCache_PutAndGet(t *testing.T) {
	cache := NewMessageCache(1024)

	key := CacheKey{Topic: "t1", Partition: 0, Offset: 0}
	msg := makeMsg("hello", 100)

	cache.Put(key, msg)

	got, ok := cache.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got.Value) != "hello" {
		t.Fatalf("expected 'hello', got %q", string(got.Value))
	}
}

func TestMessageCache_Miss(t *testing.T) {
	cache := NewMessageCache(1024)

	key := CacheKey{Topic: "t1", Partition: 0, Offset: 99}
	_, ok := cache.Get(key)
	if ok {
		t.Fatal("expected cache miss")
	}
}

func TestMessageCache_LRUEviction(t *testing.T) {
	// Cache can hold 200 bytes. Insert 3 x 100-byte messages → first should be evicted.
	cache := NewMessageCache(200)

	k1 := CacheKey{Topic: "t", Partition: 0, Offset: 0}
	k2 := CacheKey{Topic: "t", Partition: 0, Offset: 1}
	k3 := CacheKey{Topic: "t", Partition: 0, Offset: 2}

	cache.Put(k1, makeMsg("a", 100))
	cache.Put(k2, makeMsg("b", 100))
	// Cache is full at 200. Insert k3 → k1 (LRU) should be evicted.
	cache.Put(k3, makeMsg("c", 100))

	if _, ok := cache.Get(k1); ok {
		t.Fatal("k1 should have been evicted (LRU)")
	}
	if _, ok := cache.Get(k2); !ok {
		t.Fatal("k2 should still be in cache")
	}
	if _, ok := cache.Get(k3); !ok {
		t.Fatal("k3 should still be in cache")
	}
	if cache.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", cache.Len())
	}
}

func TestMessageCache_LRUEviction_AccessPromotes(t *testing.T) {
	cache := NewMessageCache(200)

	k1 := CacheKey{Topic: "t", Partition: 0, Offset: 0}
	k2 := CacheKey{Topic: "t", Partition: 0, Offset: 1}
	k3 := CacheKey{Topic: "t", Partition: 0, Offset: 2}

	cache.Put(k1, makeMsg("a", 100))
	cache.Put(k2, makeMsg("b", 100))

	// Access k1 to promote it → k2 becomes LRU
	cache.Get(k1)

	// Insert k3 → k2 (now LRU) should be evicted, not k1
	cache.Put(k3, makeMsg("c", 100))

	if _, ok := cache.Get(k1); !ok {
		t.Fatal("k1 should still be in cache (was promoted by Get)")
	}
	if _, ok := cache.Get(k2); ok {
		t.Fatal("k2 should have been evicted (was LRU)")
	}
	if _, ok := cache.Get(k3); !ok {
		t.Fatal("k3 should still be in cache")
	}
}

func TestMessageCache_UpdateExisting(t *testing.T) {
	cache := NewMessageCache(1024)

	key := CacheKey{Topic: "t", Partition: 0, Offset: 0}
	cache.Put(key, makeMsg("old", 50))
	cache.Put(key, makeMsg("new", 60))

	got, ok := cache.Get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if string(got.Value) != "new" {
		t.Fatalf("expected 'new', got %q", string(got.Value))
	}
	if cache.Size() != 60 {
		t.Fatalf("expected size 60, got %d", cache.Size())
	}
	if cache.Len() != 1 {
		t.Fatalf("expected 1 entry, got %d", cache.Len())
	}
}

func TestMessageCache_GetBatch_AllHit(t *testing.T) {
	cache := NewMessageCache(4096)

	for i := 0; i < 5; i++ {
		key := CacheKey{Topic: "t", Partition: 0, Offset: int64(i)}
		cache.Put(key, makeMsg(fmt.Sprintf("msg-%d", i), 50))
	}

	msgs, missOffset := cache.GetBatch("t", 0, 0, 5)
	if missOffset != -1 {
		t.Fatalf("expected no miss, got miss at offset %d", missOffset)
	}
	if len(msgs) != 5 {
		t.Fatalf("expected 5 messages, got %d", len(msgs))
	}
	for i, msg := range msgs {
		expected := fmt.Sprintf("msg-%d", i)
		if string(msg.Value) != expected {
			t.Fatalf("message %d: expected %q, got %q", i, expected, string(msg.Value))
		}
	}
}

func TestMessageCache_GetBatch_PartialHit(t *testing.T) {
	cache := NewMessageCache(4096)

	// Only cache offsets 0, 1, 2 — offset 3 is missing
	for i := 0; i < 3; i++ {
		key := CacheKey{Topic: "t", Partition: 0, Offset: int64(i)}
		cache.Put(key, makeMsg(fmt.Sprintf("msg-%d", i), 50))
	}

	msgs, missOffset := cache.GetBatch("t", 0, 0, 5)
	if missOffset != 3 {
		t.Fatalf("expected miss at offset 3, got %d", missOffset)
	}
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}
}

func TestMessageCache_GetBatch_CompleteMiss(t *testing.T) {
	cache := NewMessageCache(4096)

	msgs, missOffset := cache.GetBatch("t", 0, 100, 10)
	if missOffset != 100 {
		t.Fatalf("expected miss at offset 100, got %d", missOffset)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(msgs))
	}
}

func TestMessageCache_ConcurrentAccess(t *testing.T) {
	cache := NewMessageCache(100000)
	var wg sync.WaitGroup

	// 10 writers, each writing 100 entries
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func(writer int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				key := CacheKey{Topic: fmt.Sprintf("t%d", writer), Partition: 0, Offset: int64(i)}
				cache.Put(key, makeMsg("data", 10))
			}
		}(w)
	}

	// 10 readers running concurrently
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func(reader int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				key := CacheKey{Topic: fmt.Sprintf("t%d", reader), Partition: 0, Offset: int64(i)}
				cache.Get(key)
			}
		}(r)
	}

	wg.Wait()

	// No panics or data races = pass
	if cache.Len() == 0 {
		t.Fatal("expected cache to have entries after concurrent writes")
	}
}

func TestMessageCache_SizeTracking(t *testing.T) {
	cache := NewMessageCache(4096)

	cache.Put(CacheKey{Topic: "t", Partition: 0, Offset: 0}, makeMsg("a", 100))
	cache.Put(CacheKey{Topic: "t", Partition: 0, Offset: 1}, makeMsg("b", 200))

	if cache.Size() != 300 {
		t.Fatalf("expected size 300, got %d", cache.Size())
	}

	// Evict by inserting into a small cache
	small := NewMessageCache(150)
	small.Put(CacheKey{Topic: "t", Partition: 0, Offset: 0}, makeMsg("a", 100))
	small.Put(CacheKey{Topic: "t", Partition: 0, Offset: 1}, makeMsg("b", 100))

	// First entry should be evicted
	if small.Size() > 150 {
		t.Fatalf("cache size %d exceeds max 150", small.Size())
	}
}

package event

import (
	"container/list"
	"sync"
	"time"
)

// CacheKey uniquely identifies a Kafka message by topic, partition, and offset.
type CacheKey struct {
	Topic     string
	Partition int32
	Offset    int64
}

// CachedMessage holds the data of a single Kafka message stored in the cache.
type CachedMessage struct {
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Timestamp time.Time
	size      int64
}

// cacheEntry pairs a CacheKey with its CachedMessage for storage in the LRU list.
type cacheEntry struct {
	key     CacheKey
	message *CachedMessage
}

// MessageCache is a thread-safe, LRU-evicting in-memory cache for Kafka messages.
// It is shared across all lambdas on a worker.
type MessageCache struct {
	entries     map[CacheKey]*list.Element
	lruList     *list.List // front = most recently used, back = least recently used
	currentSize int64
	maxSize     int64
	mu          sync.RWMutex
}

// NewMessageCache creates a MessageCache with the given maximum size in bytes.
func NewMessageCache(maxSizeBytes int64) *MessageCache {
	return &MessageCache{
		entries: make(map[CacheKey]*list.Element),
		lruList: list.New(),
		maxSize: maxSizeBytes,
	}
}

// Get retrieves a single message from the cache. Returns the message and true
// on a hit, or nil and false on a miss. A hit promotes the entry to MRU.
func (mc *MessageCache) Get(key CacheKey) (*CachedMessage, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	elem, ok := mc.entries[key]
	if !ok {
		return nil, false
	}

	mc.lruList.MoveToFront(elem)
	return elem.Value.(*cacheEntry).message, true
}

// Put inserts a message into the cache. If the key already exists, the entry is
// updated and promoted to MRU. Evicts LRU entries if the cache exceeds maxSize.
func (mc *MessageCache) Put(key CacheKey, msg *CachedMessage) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Update existing entry
	if elem, ok := mc.entries[key]; ok {
		old := elem.Value.(*cacheEntry)
		mc.currentSize -= old.message.size
		old.message = msg
		mc.currentSize += msg.size
		mc.lruList.MoveToFront(elem)
		mc.evict()
		return
	}

	// Insert new entry
	entry := &cacheEntry{key: key, message: msg}
	elem := mc.lruList.PushFront(entry)
	mc.entries[key] = elem
	mc.currentSize += msg.size

	mc.evict()
}

// evict removes LRU entries until currentSize is at or below maxSize.
// Caller must hold mc.mu.
func (mc *MessageCache) evict() {
	for mc.currentSize > mc.maxSize && mc.lruList.Len() > 0 {
		back := mc.lruList.Back()
		if back == nil {
			break
		}
		entry := back.Value.(*cacheEntry)
		mc.lruList.Remove(back)
		delete(mc.entries, entry.key)
		mc.currentSize -= entry.message.size
	}
}

// Size returns the current cache size in bytes.
func (mc *MessageCache) Size() int64 {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.currentSize
}

// Len returns the number of entries in the cache.
func (mc *MessageCache) Len() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return len(mc.entries)
}

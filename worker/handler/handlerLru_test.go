package handler

import (
	"testing"
)

func TestLRU(t *testing.T) {
	lru := NewHandlerLRU(0)
	opts := HandlerSetOpts{
		Cm:  MockSandboxManager{},
		Lru: lru,
	}
	handlers := NewHandlerSet(opts)
	a := handlers.Get("a")

	lru.Add(a)
	if lru.Len() != 1 {
		t.Fatalf("Unexpected len: %v", lru.Len())
	}
	lru.Remove(a)
	if lru.Len() != 0 {
		t.Fatalf("Unexpected len: %v", lru.Len())
	}
	lru.Add(a)
	if lru.Len() != 1 {
		t.Fatalf("Unexpected len: %v", lru.Len())
	}
}

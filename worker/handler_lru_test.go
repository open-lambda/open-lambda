package main

import (
	"testing"
)

func TestLRU(t *testing.T) {
	lru := NewHandlerLRU(0)
	handlers := NewHandlerSet(HandlerSetOpts{lru: lru})
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

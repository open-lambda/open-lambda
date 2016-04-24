package main

import (
	"testing"
)

func TestLRU(t *testing.T) {
	handlers := NewHandlerSet(nil)
	a := handlers.Get("a")

	lru := NewHandlerLRU()
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

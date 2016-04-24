package main

import (
	"container/list"
	"sync"
)

type HandlerLRU struct {
	mutex  sync.Mutex
	hmap   map[*Handler]*list.Element
	hqueue *list.List // front is recent
}

func NewHandlerLRU() *HandlerLRU {
	return &HandlerLRU{
		hmap:   make(map[*Handler]*list.Element),
		hqueue: list.New(),
	}
}

func (lru *HandlerLRU) Len() int {
	if lru.hqueue.Len() != len(lru.hmap) {
		panic("length mismatch")
	}
	return lru.hqueue.Len()
}

func (lru *HandlerLRU) Add(handler *Handler) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	if lru.hmap[handler] != nil {
		panic("cannot double insert in LRU")
	}
	entry := lru.hqueue.PushFront(handler)
	lru.hmap[handler] = entry
}

func (lru *HandlerLRU) Remove(handler *Handler) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	entry := lru.hmap[handler]
	if entry == nil {
		panic("map entry not found")
	}
	if lru.hqueue.Remove(entry) == nil {
		panic("queue entry not found")
	}
	delete(lru.hmap, handler)
}

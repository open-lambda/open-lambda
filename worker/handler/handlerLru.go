package handler

import (
	"container/list"
	"fmt"
	"sync"
)

type HandlerLRU struct {
	mutex  sync.Mutex
	hmap   map[*Handler]*list.Element
	hqueue *list.List // front is recent
	// TODO(tyler): set hard limit to prevent new containers from starting?
	soft_limit int
	soft_cond  *sync.Cond
}

func NewHandlerLRU(soft_limit int) *HandlerLRU {
	lru := &HandlerLRU{
		hmap:       make(map[*Handler]*list.Element),
		hqueue:     list.New(),
		soft_limit: soft_limit,
	}
	lru.soft_cond = sync.NewCond(&lru.mutex)
	// TODO(tyler): start a configurable number of tasks
	go lru.Evictor()
	return lru
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

	if lru.Len() > lru.soft_limit {
		lru.soft_cond.Signal()
	}
}

func (lru *HandlerLRU) Remove(handler *Handler) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	entry := lru.hmap[handler]
	delete(lru.hmap, handler)
	if entry != nil {
		if lru.hqueue.Remove(entry) == nil {
			panic("queue entry not found")
		}
	}
}

func (lru *HandlerLRU) Evictor() {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	for {
		for lru.Len() <= lru.soft_limit {
			lru.soft_cond.Wait()
		}

		// pop off least-recently used entry
		entry := lru.hqueue.Back()
		handler := entry.Value.(*Handler)
		lru.hqueue.Remove(entry)
		delete(lru.hmap, handler)

		lru.mutex.Unlock()
		// depending on interleavings, it could also be
		// running or already stopped.
		//
		// TODO(tyler): is there a better way?
		handler.StopIfPaused()
		lru.mutex.Lock()
	}
}

func (lru *HandlerLRU) Dump() {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	fmt.Printf("LRU Entries (recent first):\n")
	for e := lru.hqueue.Front(); e != nil; e = e.Next() {
		h := e.Value.(*Handler)
		fmt.Printf("> %s\n", h.name)
	}
}

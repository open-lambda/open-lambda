package handler

import (
	"container/list"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strconv"
	"strings"
	"sync"
)

// HandlerLRU manages a list of stopped Handlers with the LRU policy.
type HandlerLRU struct {
	mutex sync.Mutex
	// use a linked list and a map to achieve a linked-map
	hmap   map[*Handler]*list.Element
	hms    *HandlerManagerSet
	hqueue *list.List // front is recent
	// TODO(tyler): set hard limit to prevent new containers from starting?
	soft_limit int
	soft_cond  *sync.Cond
	size       int
}

// NewHandlerLRU creates a HandlerLRU with a given soft_limit and starts the
// evictor in a go routine.
func NewHandlerLRU(hms *HandlerManagerSet, soft_limit int) *HandlerLRU {
	lru := &HandlerLRU{
		hmap:       make(map[*Handler]*list.Element),
		hms:        hms,
		hqueue:     list.New(),
		soft_limit: soft_limit * 1024,
		size:       0,
	}
	lru.soft_cond = sync.NewCond(&lru.mutex)
	// TODO(tyler): start a configurable number of tasks
	go lru.Evictor()
	return lru
}

// Len gets the number of Handlers in the LRU list.
func (lru *HandlerLRU) Len() int {
	if lru.hqueue.Len() != len(lru.hmap) {
		panic("length mismatch")
	}
	return lru.hqueue.Len()
}

// Add adds a Handler into the LRU list. If the resulting length of the list is
// greater than the soft limit, the evictor will be notified. It is an error to
// add a Handler to the list more than once.
func (lru *HandlerLRU) Add(handler *Handler) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	if lru.hmap[handler] != nil {
		panic("cannot double insert in LRU")
	}
	entry := lru.hqueue.PushFront(handler)
	handler.usage = handlerUsage(handler)
	lru.size += handler.usage
	lru.hmap[handler] = entry

	if lru.size > lru.soft_limit {
		lru.soft_cond.Signal()
	}
}

// Remove removes a Handler from the LRU list if exists.
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
	lru.size -= handler.usage
}

// Evictor waits on signal that the number of Handlers in the LRU list exceeds
// the soft limit, and tries to stop the LRU handlers until the limit is met.
func (lru *HandlerLRU) Evictor() {
	for {
		lru.mutex.Lock()
		for lru.size <= lru.soft_limit {
			lru.soft_cond.Wait()
		}
		lru.mutex.Unlock()
		log.Printf("EVICTING HANDLER: %v used / %v limit", lru.size, lru.soft_limit)

		// lock the HandlerManagerSet
		lru.hms.mutex.Lock()
		lru.mutex.Lock()

		if lru.hqueue.Len() == 0 {
			lru.mutex.Unlock()
			lru.hms.mutex.Unlock()
			continue
		}

		// pop off least-recently used entry
		entry := lru.hqueue.Back()
		h := entry.Value.(*Handler)
		lru.hqueue.Remove(entry)
		delete(lru.hmap, h)
		lru.size -= h.usage

		lru.mutex.Unlock()

		// modify the Handler's HandlerManager
		hm := lru.hms.hmMap[h.name]
		hm.mutex.Lock()
		hEle := hm.hElements[h]
		hm.handlers.Remove(hEle)
		delete(hm.hElements, h)
		hm.mutex.Unlock()

		lru.hms.mutex.Unlock()

		go h.nuke()

	}
}

// Dump prints the Handler names in the LRU list from most recent to least
// recent.
func (lru *HandlerLRU) Dump() {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	fmt.Printf("LRU Entries (recent first):\n")
	for e := lru.hqueue.Front(); e != nil; e = e.Next() {
		h := e.Value.(*Handler)
		fmt.Printf("> %s\n", h.name)
	}
}

func handlerUsage(handler *Handler) (usage int) {
	usagePath := path.Join(handler.sandbox.MemoryCGroupPath(), "memory.usage_in_bytes")
	buf, err := ioutil.ReadFile(usagePath)
	if err != nil {
		panic(fmt.Sprintf("get usage failed: %v", err))
	}

	str := strings.TrimSpace(string(buf[:]))
	usage, err = strconv.Atoi(str)
	if err != nil {
		panic(fmt.Sprintf("atoi failed: %v", err))
	}

	return usage
}

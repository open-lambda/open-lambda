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

// LambdaInstanceLRU manages a list of stopped LambdaInstances with the LRU policy.
type LambdaInstanceLRU struct {
	mutex sync.Mutex
	// use a linked list and a map to achieve a linked-map
	imap  map[*LambdaInstance]*list.Element
	mgr   *LambdaMgr
	queue *list.List // front is recent
	// TODO(tyler): set hard limit to prevent new containers from starting?
	soft_limit_bytes int
	soft_cond        *sync.Cond
	size             int
}

// NewLambdaInstanceLRU creates a LambdaInstanceLRU with a given soft_limit and starts the
// evictor in a go routine.
func NewLambdaInstanceLRU(mgr *LambdaMgr, soft_limit_mb int) *LambdaInstanceLRU {
	lru := &LambdaInstanceLRU{
		imap:             make(map[*LambdaInstance]*list.Element),
		mgr:              mgr,
		queue:            list.New(),
		soft_limit_bytes: soft_limit_mb * 1024 * 1024,
		size:             0,
	}
	lru.soft_cond = sync.NewCond(&lru.mutex)
	// TODO(tyler): start a configurable number of tasks
	go lru.Evictor()
	return lru
}

// Len gets the number of LambdaInstances in the LRU list.
func (lru *LambdaInstanceLRU) Len() int {
	if lru.queue.Len() != len(lru.imap) {
		panic("length mismatch")
	}
	return lru.queue.Len()
}

// Add adds a LambdaInstance into the LRU list. If the resulting length of the list is
// greater than the soft limit, the evictor will be notified. It is an error to
// add a LambdaInstance to the list more than once.
func (lru *LambdaInstanceLRU) Add(inst *LambdaInstance) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	if lru.imap[inst] != nil {
		panic("cannot double insert in LRU")
	}
	entry := lru.queue.PushFront(inst)
	inst.usage = lambdaInstanceUsage(inst)
	lru.size += inst.usage
	lru.imap[inst] = entry

	if lru.size > lru.soft_limit_bytes {
		lru.soft_cond.Signal()
	}
}

// Remove removes a LambdaInstance from the LRU list if exists.
func (lru *LambdaInstanceLRU) Remove(inst *LambdaInstance) {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	entry := lru.imap[inst]
	delete(lru.imap, inst)
	if entry != nil {
		if lru.queue.Remove(entry) == nil {
			panic("queue entry not found")
		}
	}
	lru.size -= inst.usage
}

// Evictor waits on signal that the number of LambdaInstances in the LRU list exceeds
// the soft limit, and tries to stop the LRU handlers until the limit is met.
func (lru *LambdaInstanceLRU) Evictor() {
	for {
		lru.mutex.Lock()
		for lru.size <= lru.soft_limit_bytes {
			lru.soft_cond.Wait()
		}
		lru.mutex.Unlock()
		log.Printf("EVICTING INSTANCE: %v used / %v limit", lru.size, lru.soft_limit_bytes)

		// lock the LambdaMgr
		lru.mgr.mutex.Lock()
		lru.mutex.Lock()

		if lru.queue.Len() == 0 {
			lru.mutex.Unlock()
			lru.mgr.mutex.Unlock()
			continue
		}

		// pop off least-recently used entry
		entry := lru.queue.Back()
		inst := entry.Value.(*LambdaInstance)
		lru.queue.Remove(entry)
		delete(lru.imap, inst)
		lru.size -= inst.usage

		lru.mutex.Unlock()

		// modify the LambdaInstance's LambdaManager
		hm := lru.mgr.lfuncMap[inst.name]
		hm.mutex.Lock()
		hEle := hm.listEl[inst]
		hm.instances.Remove(hEle)
		delete(hm.listEl, inst)
		hm.mutex.Unlock()

		lru.mgr.mutex.Unlock()

		go inst.sandbox.Destroy()
	}
}

// Dump prints the LambdaInstance names in the LRU list from most recent to least
// recent.
func (lru *LambdaInstanceLRU) Dump() {
	lru.mutex.Lock()
	defer lru.mutex.Unlock()

	fmt.Printf("LRU Entries (recent first):\n")
	for e := lru.queue.Front(); e != nil; e = e.Next() {
		h := e.Value.(*LambdaInstance)
		fmt.Printf("> %s\n", h.name)
	}
}

func lambdaInstanceUsage(inst *LambdaInstance) (usage int) {
	usagePath := path.Join(inst.sandbox.MemoryCGroupPath(), "memory.usage_in_bytes")
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

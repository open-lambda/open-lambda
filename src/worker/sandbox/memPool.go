package sandbox

import (
	"container/list"
	"fmt"
	"log"
	"strings"

	"github.com/open-lambda/open-lambda/ol/common"
)

type MemPool struct {
	name string

	// how much memory is being managed (includes free and allocated)
	totalMB int

	// a task listens on this, with requests to decrement memory
	// (which may block) or increment it
	memRequests chan *memReq

	// decrement requests read from memRequests that need to wait
	// for memory sit here until it's available
	memRequestsWaiting *list.List
}

type memReq struct {
	// how much we're requesting
	mb int

	// any response means the memory is allocated; the particular
	// number indicates the total remaining memory available in
	// the pool
	resp chan int
}

func NewMemPool(name string, totalMB int) *MemPool {
	pool := &MemPool{
		name:               name,
		totalMB:            totalMB,
		memRequests:        make(chan *memReq, 32),
		memRequestsWaiting: list.New(),
	}

	go pool.memTask()

	return pool
}

func (pool *MemPool) printf(format string, args ...any) {
	if common.Conf.Trace.Memory {
		msg := fmt.Sprintf(format, args...)
		log.Printf("%s [MEM POOL %s]", strings.TrimRight(msg, "\n"), pool.name)
	}
}

// this task is responsible for tracking available memory in the
// system, adding to the count when memory is released, and blocking
// requesters until enough is free
func (pool *MemPool) memTask() {
	availableMB := pool.totalMB

	for {
		req, ok := <-pool.memRequests
		if !ok {
			return
		}

		if pool.totalMB+req.mb < 0 {
			panic(fmt.Sprintf("received request for %d MB to pool of total size %d MB",
				-req.mb, pool.totalMB))
		}

		if req.mb >= 0 {
			availableMB += req.mb
			pool.printf("%d of %d MB available", availableMB, pool.totalMB)
			req.resp <- availableMB
		} else {
			pool.memRequestsWaiting.PushBack(req)
		}

		// POLICY: which requests should we serve first?
		if e := pool.memRequestsWaiting.Front(); e != nil {
			req = e.Value.(*memReq)
			// req.mb is negative
			if availableMB+req.mb >= 0 {
				pool.memRequestsWaiting.Remove(e)
				availableMB += req.mb
				pool.printf("%d of %d MB available", availableMB, pool.totalMB)
				req.resp <- availableMB
			}
		}
	}
}

// this adjusts the available memory in the pool up/down, and returns
// the remaining available after the adjustment.
//
// Available memory is kept >=0, so a negative mb may block for some
// time.
//
// Sending a mb of 0 is a reasonable use case, especially for an
// evictor (it doesn't change anything, but provides a way to monitor
// available memory).
func (pool *MemPool) adjustAvailableMB(mb int) (availableMB int) {
	req := &memReq{
		mb:   mb,
		resp: make(chan int),
	}

	pool.memRequests <- req
	return <-req.resp
}

func (pool *MemPool) getAvailableMB() (availableMB int) {
	return pool.adjustAvailableMB(0)
}

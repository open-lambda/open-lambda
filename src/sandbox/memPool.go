package sandbox

import (
	"container/list"
)

type memReq struct {
	// how much we're requesting
	mb int

	// any response means the memory is allocated; the particular
	// number indicates the total remaining memory available in
	// the pool
	resp chan int
}

type MemPool struct {
	// a task listens on this, with requests to decrement memory
	// (which may block) or increment it
	memRequests chan *memReq

	// decrement requests read from memRequests that need to wait
	// for memory sit here until it's available
	memRequestsWaiting *list.List
}

func NewMemPool(size_mb int) *MemPool {
	pool := &MemPool{
		memRequests:        make(chan *memReq, 32),
		memRequestsWaiting: list.New(),
	}

	go pool.memTask(size_mb)

	return pool
}

// this task is responsible for tracking available memory in the
// system, adding to the count when memory is released, and blocking
// requesters until enough is free
func (pool *MemPool) memTask(pool_size_mb int) {
	available_mb := pool_size_mb

	for {
		req, ok := <-pool.memRequests
		if !ok {
			return
		}

		if req.mb >= 0 {
			available_mb += req.mb
			req.resp <- available_mb
		} else {
			pool.memRequestsWaiting.PushBack(req)
		}

		if e := pool.memRequestsWaiting.Front(); e != nil {
			req = e.Value.(*memReq)
			if available_mb+req.mb >= 0 {
				pool.memRequestsWaiting.Remove(e)
				available_mb += req.mb
				req.resp <- available_mb
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
func (pool *MemPool) adjustAvailableMB(mb int) (available_mb int) {
	req := &memReq{
		mb:   mb,
		resp: make(chan int),
	}

	pool.memRequests <- req
	return <-req.resp
}

func (pool *MemPool) getAvailableMB() (available_mb int) {
	return pool.adjustAvailableMB(0)
}

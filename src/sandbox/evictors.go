package sandbox

import (
	"container/list"
	"log"

	"github.com/open-lambda/open-lambda/ol/config"
)

// we would like to have enough memory free at any time to spin up the
// following number of sandboxes, should the need arise
const FREE_SANDBOXES_GOAL = 8

// the maximum number of evictions we'll do concurrently
const CONCURRENT_EVICTIONS = 8

type SOCKEvictor struct {
	// used to track memory pressure
	mem *MemPool

	// how we're notified of containers starting, pausing, etc
	events chan EvictorEvent

	// state queues (each Sandbox is on at most one of these)
	running  *list.List
	paused   *list.List
	evicting *list.List

	// map Sandbox ID to the List/Element position in a state queue
	stateMap map[string]*ListLocation
}

type EvictorEvent struct {
	evType SandboxEventType
	sb     Sandbox
}

type ListLocation struct {
	*list.List
	*list.Element
}

func NewSOCKEvictor(sbPool *SOCKPool) *SOCKEvictor {
	e := &SOCKEvictor{
		mem:      sbPool.mem,
		events:   make(chan EvictorEvent, 64),
		running:  list.New(),
		paused:   list.New(),
		evicting: list.New(),
		stateMap: make(map[string]*ListLocation),
	}

	sbPool.AddListener(e.Event)
	go e.Run()

	return e
}

func (evictor *SOCKEvictor) Event(evType SandboxEventType, sb Sandbox) {
	evictor.events <- EvictorEvent{evType, sb}
}

// move Sandbox to a given queue, removing from previous (if necessary).
// a move to nil is just a delete.
//
// a Sandbox cannot be moved from evicting to another queue (only to nil);
// requests attempting to do so are quietly ignored
func (evictor *SOCKEvictor) move(sb Sandbox, target *list.List) {
	// remove from previous queue if necessary
	prev := evictor.stateMap[sb.ID()]
	if prev != nil {
		// you cannot move off evicting to a live queue
		if prev.List == evictor.evicting && target != nil {
			return
		}

		prev.List.Remove(prev.Element)
	}

	// add to new queue
	if target != nil {
		if target != nil {
			element := target.PushBack(sb)
			evictor.stateMap[sb.ID()] = &ListLocation{target, element}
		}
	} else {
		delete(evictor.stateMap, sb.ID())
	}
}

func (evictor *SOCKEvictor) nextEvent(block bool) *EvictorEvent {
	if block {
		event := <-evictor.events
		return &event
	}

	select {
	case event := <-evictor.events:
		return &event
	default:
		return nil
	}
}

// update state based on messages sent to this task.  this may be
// stale, but correctness doesn't depend on freshness.
//
// blocks until there's at least one event
func (evictor *SOCKEvictor) updateState() {
	event := evictor.nextEvent(true)

	// update state based on incoming messages
	for event != nil {
		// add list to appropriate queue
		sb := event.sb

		switch event.evType {
		case evCreate:
			evictor.move(sb, evictor.running)
		case evUnpause:
			evictor.move(sb, evictor.running)
		case evPause:
			evictor.move(sb, evictor.paused)
		case evDestroy:
			evictor.move(sb, nil)
		default:
			log.Printf("Unknown event: %v", event.evType)
		}

		event = evictor.nextEvent(false)
	}
}

// evict whatever SB is at the front of the queue, assumes
// queue is not empty
func (evictor *SOCKEvictor) evictFront(queue *list.List) {
	front := queue.Front()
	sb := front.Value.(Sandbox)

	log.Printf("Evict Sandbox %v", sb.ID())

	// destroy async (we'll know when it's done, because
	// we'll see a evDestroy event later on our chan)
	go sb.Destroy()
	evictor.move(sb, evictor.evicting)
}

func (evictor *SOCKEvictor) doEvictions() {
	// how many sandboxes could we spin up, given available mem?
	memLimitMB := config.Conf.Sock_cgroups.Max_mem_mb
	freeSandboxes := evictor.mem.getAvailableMB() / memLimitMB

	// how many shoud we try to evict?
	//
	// TODO: consider counting in-flight evictions.  This will be
	// a bit tricky, as the evictions may be of sandboxes in paused
	// states with reduced memory limits
	evictCount := FREE_SANDBOXES_GOAL - freeSandboxes

	evictCap := CONCURRENT_EVICTIONS - evictor.evicting.Len()
	if evictCap < evictCount {
		evictCount = evictCap
	}

	// try evicting the desired number, starting with the paused queue
	for evictCount > 0 && evictor.paused.Len() > 0 {
		evictor.evictFront(evictor.paused)
		evictCount -= 1
	}

	// we don't like to evict running containers, because that
	// interrupts requests, but we do if necessary to keep the
	// system moving (what if all lambdas hanged forever?)
	//
	// TODO: create some parameters to better control eviction in
	// this state
	if freeSandboxes <= 0 && evictor.evicting.Len() == 0 {
		if evictor.running.Len() > 0 {
			evictor.evictFront(evictor.running)
		}
	}
}

func (evictor *SOCKEvictor) Run() {
	// map container ID to the element that is on one of the lists

	for {
		// blocks until there's at least one update
		evictor.updateState()

		// select 0 or more sandboxes to evict (policy), then
		// .Destroy them (mechanism)
		evictor.doEvictions()
	}
}

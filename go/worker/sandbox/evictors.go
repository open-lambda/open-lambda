package sandbox

import (
	"container/list"
	"fmt"
	"log"
	"strings"

	"github.com/open-lambda/open-lambda/ol/common"
)

// we would like 20% of the pool to be free for new containers.  the
// evictor can only run if there's enough memory for two containers.
// if there are only 2, our goal is to have free mem for on container.
// 20% only applies to containers in excess of 2.
const FREE_SANDBOXES_PERCENT_GOAL = 20

// the maximum number of evictions we'll do concurrently
const CONCURRENT_EVICTIONS = 8

type SOCKEvictor struct {
	// used to track memory pressure
	mem *MemPool

	// how we're notified of containers starting, pausing, etc
	events chan SandboxEvent

	// Sandbox ID => prio.  we ALWAYS evict lower priority before higher priority
	//
	// A Sandbox's priority is 2*NUM_CHILDEN, +1 if Unpaused.
	// Thus, we'll prefer to evict paused (idle) sandboxes with no
	// children.  Under pressure, we'll evict running sandboxes
	// (this will surface an error to the end user).  We'll never
	// invoke from priority 2+ (i.e., those with at least one
	// child), as there is no benefit to evicting Sandboxes with
	// live children (we can't reclaim memory until all
	// descendents exit)
	priority map[string]int

	// state queues (each Sandbox is on at most one of these)
	prioQueues []*list.List
	evicting   *list.List

	// Sandbox ID => List/Element position in a state queue
	stateMap map[string]*ListLocation
}

type ListLocation struct {
	*list.List
	*list.Element
}

func NewSOCKEvictor(sbPool *SOCKPool) *SOCKEvictor {
	// level 0: no children, paused
	// level 1: no children, unpaused
	// level 2: children
	prioQueues := make([]*list.List, 3, 3)
	for i := 0; i < len(prioQueues); i++ {
		prioQueues[i] = list.New()
	}

	e := &SOCKEvictor{
		mem:        sbPool.mem,
		events:     make(chan SandboxEvent, 64),
		priority:   make(map[string]int),
		prioQueues: prioQueues,
		evicting:   list.New(),
		stateMap:   make(map[string]*ListLocation),
	}

	sbPool.AddListener(e.Event)
	go e.Run()

	return e
}

func (evictor *SOCKEvictor) Event(evType SandboxEventType, sb Sandbox) {
	evictor.events <- SandboxEvent{evType, sb}
}

// move Sandbox to a given queue, removing from previous (if necessary).
// a move to nil is just a delete.
func (evictor *SOCKEvictor) move(sb Sandbox, target *list.List) {
	// remove from previous queue if necessary
	prev := evictor.stateMap[sb.ID()]
	if prev != nil {
		prev.List.Remove(prev.Element)
	}

	// add to new queue
	if target != nil {
		element := target.PushBack(sb)
		evictor.stateMap[sb.ID()] = &ListLocation{target, element}
	} else {
		delete(evictor.stateMap, sb.ID())
	}
}

func (evictor *SOCKEvictor) nextEvent(block bool) *SandboxEvent {
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

func (_ *SOCKEvictor) printf(format string, args ...any) {
	if common.Conf.Trace.Evictor {
		msg := fmt.Sprintf(format, args...)
		log.Printf("%s [EVICTOR]", strings.TrimRight(msg, "\n"))
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
		sb := event.SB
		prio := evictor.priority[sb.ID()]

		switch event.EvType {
		case EvCreate:
			if prio != 0 {
				panic(fmt.Sprintf("Sandboxes should be at prio 0 upon EvCreate event but it was %d for %d", prio, sb.ID()))
			}
			prio += 1
		case EvUnpause:
			prio += 1
		case EvPause:
			prio -= 1
		case EvFork:
			prio += 2
		case EvChildExit:
			prio -= 2
		case EvDestroy, EvDestroyIgnored:
		default:
			evictor.printf("Unknown event: %v", event.EvType)
		}

		evictor.printf("Evictor: Sandbox %v priority goes to %d", sb.ID(), prio)
		if prio < 0 {
			panic(fmt.Sprintf("priority should never go negative, but it went to %d for sandbox %d", prio, sb.ID()))
			panic("priority should never go negative")
		}

		if event.EvType == EvDestroy {
			evictor.move(sb, nil)
			delete(evictor.priority, sb.ID())
		} else {
			evictor.priority[sb.ID()] = prio
			// saturate prio based on number of queues
			if prio >= len(evictor.prioQueues) {
				prio = len(evictor.prioQueues) - 1
			}

			evictor.move(sb, evictor.prioQueues[prio])
		}

		event = evictor.nextEvent(false)
	}
}

// evict whatever SB is at the front of the queue, assumes
// queue is not empty
func (evictor *SOCKEvictor) evictFront(queue *list.List, force bool) {
	front := queue.Front()
	sb := front.Value.(Sandbox)

	evictor.printf("Evict Sandbox %v", sb.ID())
	evictor.move(sb, evictor.evicting)

	// destroy async (we'll know when it's done, because
	// we'll see a evDestroy event later on our chan)
	go func() {
		t := common.T0("evict")
		if force {
			sb.Destroy("forced eviction")
		} else {
			sb.DestroyIfPaused("idle eviction")
		}
		t.T1()
	}()
}

// POLICY: how should we select a victim?
func (evictor *SOCKEvictor) doEvictions() {
	memLimitMB := common.Conf.Limits.Mem_mb

	// how many sandboxes could we spin up, given available mem?
	freeSandboxes := evictor.mem.getAvailableMB() / memLimitMB

	// how many sandboxes would we like to be able to spin up,
	// without waiting for more memory?
	freeGoal := 1 + ((evictor.mem.totalMB/memLimitMB)-2)*FREE_SANDBOXES_PERCENT_GOAL/100

	// how many shoud we try to evict?
	//
	// TODO: consider counting in-flight evictions.  This will be
	// a bit tricky, as the evictions may be of sandboxes in paused
	// states with reduced memory limits
	evictCount := freeGoal - freeSandboxes

	evictCap := CONCURRENT_EVICTIONS - evictor.evicting.Len()
	if evictCap < evictCount {
		evictCount = evictCap
	}

	// try evicting the desired number, starting with the paused queue
	for evictCount > 0 && evictor.prioQueues[0].Len() > 0 {
		evictor.evictFront(evictor.prioQueues[0], false)
		evictCount -= 1
	}

	// we don't like to evict running containers, because that
	// interrupts requests, but we do if necessary to keep the
	// system moving (what if all lambdas hanged forever?)
	//
	// TODO: create some parameters to better control eviction in
	// this state
	if freeSandboxes <= 0 && evictor.evicting.Len() == 0 {
		evictor.printf("WARNING!  Critically low on memory, so evicting an active Sandbox")
		if evictor.prioQueues[1].Len() > 0 {
			evictor.evictFront(evictor.prioQueues[1], true)
		}
	}

	// we never evict from prioQueues[2+], because those have
	// descendents with lower priority that should be evicted
	// first
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

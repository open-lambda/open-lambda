package sandbox

// this layer can wrap any sandbox, and provides several (mostly) safety features:
// 1. it prevents concurrent calls to Sandbox functions that modify the Sandbox
// 2. it automatically destroys unhealthy sandboxes (it is considered unhealthy after returnning any error)
// 3. calls on a destroyed sandbox just return a DEAD_SANDBOX error (no harm is done)
// 4. suppresses Pause calls to already paused Sandboxes, and similar for Unpause calls.
// 5. it traces all calls

import (
	"fmt"
	"log"
	"net/http/httputil"
	"strings"
	"sync"

	"github.com/open-lambda/open-lambda/ol/common"
)

type safeSandbox struct {
	Sandbox

	sync.Mutex
	paused        bool
	dead          error
	eventHandlers []SandboxEventFunc
}

// caller is responsible for calling startNotifyingListeners after
// init is complete.
//
// the rational is that we might need to do some setup (e.g., forking)
// after a safeSandbox is created, and that setup may fail.  We never
// want to notify listeners about a Sandbox that isn't ready to go.
// E.g., would be problematic if an evictor (which is listening) were
// to try to evict concurrently with us creating processes in the
// Sandbox as part of setup.
func newSafeSandbox(innerSB Sandbox) *safeSandbox {
	sb := &safeSandbox{
		Sandbox: innerSB,
	}

	return sb
}

func (sb *safeSandbox) startNotifyingListeners(eventHandlers []SandboxEventFunc) {
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	sb.eventHandlers = eventHandlers
	sb.event(EvCreate)
}

// like regular printf, with suffix indicating which sandbox produced the message
func (sb *safeSandbox) printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s [SB %s]", strings.TrimRight(msg, "\n"), sb.Sandbox.ID())
}

// propogate event to anybody who signed up to listen (e.g., an evictor)
func (sb *safeSandbox) event(evType SandboxEventType) {
	for _, handler := range sb.eventHandlers {
		handler(evType, sb)
	}
}

// assumes lock is already held
func (sb *safeSandbox) destroyOnErr(funcName string, origErr error) {
	if origErr != nil {
		sb.printf("Destroy() due to %v", origErr)
		sb.Sandbox.Destroy(fmt.Sprintf("%s returned %s", funcName, origErr))
		sb.dead = SandboxDeadError(fmt.Sprintf("Sandbox previously killed automatically after %s returned %s", funcName, origErr))

		// let anybody interested know this died
		sb.event(EvDestroy)
	}
}

func (sb *safeSandbox) Destroy(reason string) {
	sb.printf("Destroy()")
	t := common.T0("Destroy()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()

	if sb.dead != nil {
		return
	}

	sb.Sandbox.Destroy(reason)
	// TODO: allow message to be passed in (so we can blame cache eviction, for example)
	sb.dead = SandboxDeadError(fmt.Sprintf("Sandbox previously killed exlicitly by Destroy(reason=%s) call", reason))

	// let anybody interested know this died
	sb.event(EvDestroy)
}

func (sb *safeSandbox) Pause() (err error) {
	sb.printf("Pause()")
	t := common.T0("Pause()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()

	if sb.dead != nil {
		return sb.dead
	} else if sb.paused {
		return nil
	}

	if err := sb.Sandbox.Pause(); err != nil {
		sb.destroyOnErr("Pause", err)
		return err
	}

	sb.event(EvPause)
	sb.paused = true
	return nil
}

func (sb *safeSandbox) Unpause() (err error) {
	sb.printf("Unpause()")
	t := common.T0("Unpause()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()

	if sb.dead != nil {
		return sb.dead
	} else if !sb.paused {
		return nil
	}

	if err := sb.Sandbox.Unpause(); err != nil {
		sb.destroyOnErr("Unpause", err)
		return err
	}

	sb.event(EvUnpause)
	sb.paused = false
	return nil
}

func (sb *safeSandbox) HttpProxy() (p *httputil.ReverseProxy, err error) {
	sb.printf("Channel()")
	t := common.T0("Channel()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()

	if sb.dead != nil {
		return nil, sb.dead
	}

	p, err = sb.Sandbox.HTTPProxy()
	if err != nil {
		sb.destroyOnErr("HttpProxy", err) // TODO: rename if/when HttpProxy is replaced
	}
	return p, err
}

// fork (as a private method) doesn't cleanup parent sb if fork fails
func (sb *safeSandbox) fork(dst Sandbox) (err error) {
	sb.printf("fork(SB %v)", dst.ID())
	t := common.T0("fork()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()

	if sb.dead != nil {
		return sb.dead
	}

	if err := sb.Sandbox.fork(dst); err != nil {
		return err
	}

	sb.event(EvFork)
	return nil
}

func (sb *safeSandbox) childExit(child Sandbox) {
	sb.printf("childExit(SB %v)", child.ID())
	t := common.T0("childExit()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()

	// after a Sandbox is Destroyed, we keep sending it childExit
	// calls (so it can know when the ref count hits zero and we
	// can return the memory to the pool), but we stop notifying
	// listeners of this

	sb.Sandbox.childExit(child)

	if sb.dead == nil {
		sb.event(EvChildExit)
	}
}

func (sb *safeSandbox) Status(key SandboxStatus) (stat string, err error) {
	sb.printf("Status(%d)", key)
	t := common.T0("Status()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()

	if sb.dead != nil {
		return "", sb.dead
	}

	stat, err = sb.Sandbox.Status(key)
	if err != nil && err != STATUS_UNSUPPORTED {
		sb.destroyOnErr("Status", err)
	}
	return stat, err
}

func (sb *safeSandbox) DebugString() string {
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()

	if sb.dead != nil {
		return fmt.Sprintf("SANDBOX %s: DEAD (%s)\n", sb.Sandbox.ID(), sb.dead.Error())
	}

	return sb.Sandbox.DebugString()
}

package sandbox

// this layer can wrap any sandbox, and provides several (mostly) safety features:
// 1. it prevents concurrent calls to Sandbox functions that modify the Sandbox
// 2. it automatically destroys unhealthy sandboxes (it is considered unhealthy after returnning any error)
// 3. calls on a destroyed sandbox just return a DEAD_SANDBOX error (no harm is done)
// 4. it traces all calls

import (
	"fmt"
	"log"
	"net/http/httputil"
	"strings"
	"sync"

	"github.com/open-lambda/open-lambda/ol/stats"
)

type safeSandbox struct {
	Sandbox
	sync.Mutex
	dead          bool
	eventHandlers []SandboxEventFunc
}

func newSafeSandbox(innerSB Sandbox, eventHandlers []SandboxEventFunc) *safeSandbox {
	sb := &safeSandbox{
		Sandbox:       innerSB,
		eventHandlers: eventHandlers,
	}

	sb.event(EvCreate)

	return sb
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
func (sb *safeSandbox) destroyOnErr(origErr error) {
	if origErr != nil {
		sb.printf("Destroy() due to %v", origErr)
		sb.Sandbox.Destroy()
		sb.dead = true

		// let anybody interested know this died
		sb.event(EvDestroy)
	}
}

func (sb *safeSandbox) Destroy() {
	sb.printf("Destroy()")
	t := stats.T0("Destroy()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()

	if !sb.dead {
		sb.Sandbox.Destroy()
		sb.dead = true

		// let anybody interested know this died
		sb.event(EvDestroy)
	}
}

func (sb *safeSandbox) Pause() (err error) {
	sb.printf("Pause()")
	t := stats.T0("Pause()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err)
		if err == nil {
			// let anybody interested we paused
			sb.event(EvPause)
		}
	}()

	return sb.Sandbox.Pause()
}

func (sb *safeSandbox) Unpause() (err error) {
	sb.printf("Unpause()")
	t := stats.T0("Unpause()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err)
		if err == nil {
			// let anybody interested we paused
			sb.event(EvUnpause)
		}
	}()

	return sb.Sandbox.Unpause()
}

func (sb *safeSandbox) HttpProxy() (p *httputil.ReverseProxy, err error) {
	sb.printf("Channel()")
	t := stats.T0("Channel()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return nil, DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err)
	}()

	return sb.Sandbox.HttpProxy()
}

func (sb *safeSandbox) fork(dst Sandbox) (err error) {
	sb.printf("fork(%v)", dst)
	t := stats.T0("fork()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err)
	}()

	return sb.Sandbox.fork(dst)
}

func (sb *safeSandbox) DebugString() string {
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return fmt.Sprintf("SANDBOX %s: DEAD\n", sb.Sandbox.ID())
	}
	return sb.Sandbox.DebugString()
}

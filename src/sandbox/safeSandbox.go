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

	"github.com/open-lambda/open-lambda/ol/common"
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
func (sb *safeSandbox) destroyOnErr(origErr error, allowed []error) {
	if origErr != nil {
		for _, err := range allowed {
			if origErr == err {
				return
			}
		}

		sb.printf("Destroy() due to %v", origErr)
		sb.Sandbox.Destroy()
		sb.dead = true

		// let anybody interested know this died
		sb.event(EvDestroy)
	}
}

func (sb *safeSandbox) Destroy() {
	sb.printf("Destroy()")
	t := common.T0("Destroy()")
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
	t := common.T0("Pause()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err, []error{})
		if err == nil {
			// let anybody interested we paused
			sb.event(EvPause)
		}
	}()

	return sb.Sandbox.Pause()
}

func (sb *safeSandbox) Unpause() (err error) {
	sb.printf("Unpause()")
	t := common.T0("Unpause()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err, []error{})
		if err == nil {
			// let anybody interested we paused
			sb.event(EvUnpause)
		}
	}()

	return sb.Sandbox.Unpause()
}

func (sb *safeSandbox) HttpProxy() (p *httputil.ReverseProxy, err error) {
	sb.printf("Channel()")
	t := common.T0("Channel()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return nil, DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err, []error{})
	}()

	return sb.Sandbox.HttpProxy()
}

// fork (as a private method) doesn't cleanup parent sb if fork fails
func (sb *safeSandbox) fork(dst Sandbox) (err error) {
	sb.printf("fork(SB %v)", dst.ID())
	t := common.T0("fork()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return DEAD_SANDBOX
	}

	return sb.Sandbox.fork(dst)
}

func (sb *safeSandbox) childExit(child Sandbox) {
	sb.printf("childExit(SB %v)", child.ID())
	t := common.T0("childExit()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()

	sb.Sandbox.childExit(child)
}

func (sb *safeSandbox) Status(key SandboxStatus) (stat string, err error) {
	sb.printf("Status(%d)", key)
	t := common.T0("Status()")
	defer t.T1()
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return "", DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err, []error{STATUS_UNSUPPORTED})
	}()
	return sb.Sandbox.Status(key)
}

func (sb *safeSandbox) DebugString() string {
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return fmt.Sprintf("SANDBOX %s: DEAD\n", sb.Sandbox.ID())
	}
	return sb.Sandbox.DebugString()
}

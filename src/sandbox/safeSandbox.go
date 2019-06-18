package sandbox

// this layer can wrap any sandbox, and provides several safety features:
// 1. it prevents concurrent calls to Sandbox functions that modify the Sandbox
// 2. it automatically destroys unhealthy sandboxes
// 3. calls on a destroyed sandbox just return a DEAD_SANDBOX error (no harm is done)

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

type SockError string

const (
	DEAD_SANDBOX = SockError("Sandbox has died")
)

func (e SockError) Error() string {
	return string(e)
}

type safeSandbox struct {
	Sandbox
	sync.Mutex
	dead bool
}

func (sb *safeSandbox) printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s [SB %s]", strings.TrimRight(msg, "\n"), sb.Sandbox.ID())
}

func (sb *safeSandbox) destroyOnErr(origErr error) {
	if origErr != nil {
		sb.printf("Destroy() due to %v", origErr)
		sb.Sandbox.Destroy()
		sb.dead = true
	}
}

func (sb *safeSandbox) Destroy() {
	sb.printf("Destroy()")
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()

	if !sb.dead {
		sb.Sandbox.Destroy()
		sb.dead = true
	}
}

func (sb *safeSandbox) Pause() (err error) {
	sb.printf("Pause()")
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err)
	}()

	return sb.Sandbox.Pause()
}

func (sb *safeSandbox) Unpause() (err error) {
	sb.printf("Unpause()")
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err)
	}()

	return sb.Sandbox.Unpause()
}

func (sb *safeSandbox) Channel() (tr *http.Transport, err error) {
	sb.printf("Channel()")
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return nil, DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err)
	}()

	return sb.Sandbox.Channel()
}

func (sb *safeSandbox) MemUsageKB() (kb int, err error) {
	sb.printf("MemUsageKB()")
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return 0, DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err)
	}()

	return sb.Sandbox.MemUsageKB()
}

func (sb *safeSandbox) fork(dst Sandbox, imports []string, isLeaf bool) (err error) {
	sb.printf("fork(dst, %v, %v)", imports, isLeaf)
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return DEAD_SANDBOX
	}
	defer func() {
		sb.destroyOnErr(err)
	}()

	return sb.Sandbox.fork(dst, imports, isLeaf)
}

func (sb *safeSandbox) DebugString() string {
	sb.Mutex.Lock()
	defer sb.Mutex.Unlock()
	if sb.dead {
		return fmt.Sprintf("SANDBOX %s: DEAD\n", sb.Sandbox.ID())
	}
	return sb.Sandbox.DebugString()
}

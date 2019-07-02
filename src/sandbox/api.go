package sandbox

import (
	"net/http/httputil"
)

type SandboxPool interface {
	// Create a new, unpaused sandbox
	//
	// parent: a sandbox to fork from (may be nil, and some SandboxPool's don't support not nil)
	// isLeaf: true iff this is not being created as a sandbox we can fork later
	// codeDir: directory where lambda code exists
	// scratchDir: directory where handler code can write (caller is responsible for creating and deleting)
	// imports: Python modules that will be used (this is a hint)
	Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, imports []string) (sb Sandbox, err error)

	// All containers must be deleted before this is called, or it
	// will hang
	Cleanup()

	// handler will be called whenever a Sandbox is created, deleted, etc
	AddListener(handler SandboxEventFunc)

	DebugString() string
}

/*
Defines interfaces for sandboxing methods (e.g., container, unikernel).
Currently, only containers are supported. No need to increase complexity by
generalizing for other sandboxing methods before they are implemented.
*/
type Sandbox interface {
	// Return ID of the container.
	ID() string

	// Frees all resources associated with the container.
	// Any errors are logged, but not propagated.
	Destroy()

	// Pauses the container.
	Pause() error

	// Unpauses the container.
	Unpause() error

	// Communication channel to forward requests.
	HttpProxy() (*httputil.ReverseProxy, error)

	// How much memory does the cgroup report for this container?
	MemUsageKB() (int, error)

	// Represent state as a multi-line string
	DebugString() string

	// Optional interface for forking across sandboxes.  Sandbox may
	fork(dst Sandbox) error
}

// reference to function that will be called by sandbox pool upon key
// events
type SandboxEventFunc func(SandboxEventType, Sandbox)

type SandboxEventType int

const (
	EvCreate  SandboxEventType = iota
	EvDestroy                  = iota
	EvPause                    = iota
	EvUnpause                  = iota
)

type SandboxEvent struct {
	EvType SandboxEventType
	SB     Sandbox
}

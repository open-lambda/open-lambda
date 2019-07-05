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
	// deps: packages and modules needed by the Sandbox
	Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta) (sb Sandbox, err error)

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

	// Frees Sandbox resources (e.g., cgroups, processes), but
	// leaves Sandbox structure intact.
	//
	// There is no harm in calling any function on the Sandbox
	// after Destroy.  Functions like ID() will still work, and
	// those that are able to return errors will be no-ops,
	// returning DEAD_SANDBOX.
	Destroy()

	// Make processes in the container non-schedulable
	Pause() error

	// Make processes in the container schedulable
	Unpause() error

	// Communication channel to forward requests.
	HttpProxy() (*httputil.ReverseProxy, error)

	// Represent state as a multi-line string
	DebugString() string

	// Lookup a particular stat
	Status(SandboxStatus) (string, error)

	// Optional interface for forking across sandboxes.
	fork(dst Sandbox) error
}

type SandboxMeta struct {
	Installs   []string
	Imports    []string
	MemLimitMB int
}

type SockError string

const (
	DEAD_SANDBOX       = SockError("Sandbox has died")
	FORK_FAILED        = SockError("Fork from parent Sandbox failed")
	STATUS_UNSUPPORTED = SockError("Argument to Status(...) unsupported by this Sandbox")
)

func (e SockError) Error() string {
	return string(e)
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

type SandboxStatus int

const (
	StatusMemFailures SandboxStatus = iota // boolean
)

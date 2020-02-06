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
	// meta: details about installs, imports, etc.  Will be populated with defaults if not specified
	Create(parent Sandbox, isLeaf bool, codeDir, scratchDir string, meta *SandboxMeta) (sb Sandbox, err error)

	// blocks until all Sandboxes are deleted, so caller must
	// either delete them before this call, or from another asyncronously
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

	// Lookup metadata that Sandbox was initialized with (static over time)
	Meta() *SandboxMeta

	// Lookup a particular stat (changes over time)
	Status(SandboxStatus) (string, error)

	// Represent state as a multi-line string
	DebugString() string

	// Optional interface for creating processes in children, and
	// being notified when they die
	fork(dst Sandbox) error

	// Child calls this on parent to notify of child Destroy
	childExit(child Sandbox)
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

// reference to function that will be called by sandbox pool upon key
// events
type SandboxEventFunc func(SandboxEventType, Sandbox)

// Listeners are guaranteed that the first event seen is an EvCreate,
// and the last is an EvDestroy.  Internally, EvChildExit may occur
// after Destroy, but listeners don't see it.
//
// Listeners only see events after they occur successfully.
//
// EvPause and EvUnpause are only seen if they have an effect (e.g.,
// calling .Pause on an already-paused container will not trigger an
// event for the listeners).
type SandboxEventType int

const (
	EvCreate    SandboxEventType = iota
	EvDestroy                    = iota
	EvPause                      = iota
	EvUnpause                    = iota
	EvFork                       = iota
	EvChildExit                  = iota
)

type SandboxEvent struct {
	EvType SandboxEventType
	SB     Sandbox
}

type SandboxStatus int

const (
	StatusMemFailures SandboxStatus = iota // boolean
)

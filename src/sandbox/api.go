package sandbox

import (
	"net/http"
)

type SandboxPool interface {
	// Create a new sandbox
	//
	// codeDir: directory where lambda code exists
	// workingDir: directory in which a scratch dir for the sandbox will be allocated
	// imports: Python modules that will be used (this is a hint)
	Create(codeDir, workingDir string, imports []string) (Sandbox, error)
	Cleanup()
}

/*
Defines interfaces for sandboxing methods (e.g., container, unikernel).
Currently, only containers are supported. No need to increase complexity by
generalizing for other sandboxing methods before they are implemented.
*/
type Sandbox interface {
	// Return ID of the container.
	ID() string

	// Directory used by the worker to communicate with container.
	HostDir() string

	// Frees all resources associated with the container.
	// Any errors are logged, but not propagated.
	Destroy()

	// Pauses the container.
	Pause() error

	// Unpauses the container.
	Unpause() error

	// Communication channel to forward requests.
	Channel() (*http.Transport, error)

	// How much memory does the cgroup report for this container?
	MemUsageKB() (int, error)

	// Optional interface for forking across sandboxes
	fork(dst Sandbox, imports []string, isLeaf bool) error
}

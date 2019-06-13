package sandbox

import (
	"net/http"
)

/*
Defines interfaces for sandboxing methods (e.g., container, unikernel).
Currently, only containers are supported. No need to increase complexity by
generalizing for other sandboxing methods before they are implemented.
*/

type Channel struct {
	Url       string
	Transport http.Transport
}

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
	Channel() (*Channel, error)

	// How much memory does the cgroup report for this container?
	MemUsageKB() (int, error)

	// Directory used by the worker to communicate with container.
	HostDir() string

	// Return recent logs for the container.
	Logs() (string, error)
}

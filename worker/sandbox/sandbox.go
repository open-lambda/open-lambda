package sandbox

import (
	"net/http"
)

/*

Defines the sandbox interface. This interface abstracts all mechanisms
surrounding a given sandbox type (Docker container, cgroup, etc).

*/

import "github.com/open-lambda/open-lambda/worker/handler/state"

type SandboxChannel struct {
	Url       string
	Transport http.Transport
}

type Sandbox interface {
	// Starts a given sandbox
	Start() error

	// Stops a given sandbox
	Stop() error

	// Pauses a given sandbox
	Pause() error

	// Unpauses a given sandbox
	Unpause() error

	// Frees all resources associated with a given lambda
	// (will stop if needed)
	Remove() error

	// Return recent log output for sandbox
	Logs() (string, error)

	// Get current state
	State() (state.HandlerState, error)

	// What communication channel can we use to forward requests?
	Channel() (*SandboxChannel, error)
}

type ContainerSandbox interface {
	Sandbox

	// Put the passed process into the cgroups of the container
	CGroupEnter(pid string) error

	// PID of a process in the container's namespaces (for joining)
	NSPid() string

	// Install the passed pip packages to the container.
	DoInstalls() error
}

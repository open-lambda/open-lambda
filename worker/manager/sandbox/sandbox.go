package sandbox

/*

Defines the sandbox interface. This interface abstracts all mechanisms
surrounding a given sandbox type (Docker container, cgroup, etc).

*/

import "github.com/open-lambda/open-lambda/worker/handler/state"

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

	// What port can we use to forward requests?
	Port() (string, error)
}

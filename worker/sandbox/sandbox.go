package sandbox

import "github.com/open-lambda/open-lambda/worker/handler/state"

type Sandbox interface {
	// Runs any preperation to get the sandbox ready to run
	// Returns the current state of the sandbox
	MakeReady() error

	// Starts a given sandbox
	Start() error

	// Pauses a given sandbox
	Pause() error

	// Unpauses a given sandbox
	Unpause() error

	// Stops a given sandbox
	Stop() error

	// Frees all resources associated with a given lambda
	// Will stop if needed
	Remove() error

	// Return recent log output for sandbox
	Logs() (string, error)

	// Get current state
	State() (state.HandlerState, error)

	// What port can we use to forward requests?
	Port() (string, error)
}

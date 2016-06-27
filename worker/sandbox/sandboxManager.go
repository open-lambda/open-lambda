package sandbox

import "github.com/open-lambda/open-lambda/worker/handler/state"

type SandboxManager interface {

	// Runs any preperation to get the sandbox ready to run
	// Returns the current state of the sandbox
	MakeReady(name string) (SandboxInfo, error)

	// Returns info on the current sandbox
	GetInfo(name string) (SandboxInfo, error)

	// Starts a given sandbox
	Start(name string) error

	// Pauses a given sandbox
	Pause(name string) error

	// Unpauses a given sandbox
	Unpause(name string) error

	// Stops a given sandbox
	Stop(name string) error

	// Frees all resources associated with a given lambda
	// Will stop if needed
	Remove(name string) error

	// Return recent log output for sandbox
	Logs(name string) (string, error)
}

type SandboxInfo struct {
	State state.HandlerState
	Port  string
}

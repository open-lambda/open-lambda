package container

import "github.com/tylerharter/open-lambda/worker/handler/state"

type ContainerManager interface {

	// Runs any preperation to get the container ready to run
	// Returns the current state of the container
	MakeReady(name string) (ContainerInfo, error)

	// Returns info on the current container
	GetInfo(name string) (ContainerInfo, error)

	// Starts a given container
	Start(name string) error

	// Pauses a given container
	Pause(name string) error

	// Unpauses a given container
	Unpause(name string) error

	// Stops a given container
	Stop(name string) error

	// Frees all resources associated with a given lambda
	// Will stop if needed
	Remove(name string) error

	// Return recent log output for container
	Logs(name string) (string, error)
}

type ContainerInfo struct {
	State state.HandlerState
	Port  string
}

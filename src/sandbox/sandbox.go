package sandbox

import (
	"net/http"
)

/*
Defines interfaces for sandboxing methods (e.g., container, unikernel).
Currently, only containers are supported. No need to increase complexity by
generalizing for other sandboxing methods before they are implemented.
*/

import (
	"github.com/open-lambda/open-lambda/ol/handler/state"
)

const OLCGroupName = "openlambda"

var CGroupList []string = []string{"blkio", "cpu", "devices", "freezer", "hugetlb", "memory", "perf_event", "systemd"}

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

	// Return recent logs for the container.
	Logs() (string, error)

	// Get current state of the container.
	State() (state.HandlerState, error)

	// Communication channel to forward requests.
	Channel() (*Channel, error)

	// Path to this container's memory cgroup for accounting.
	MemoryCGroupPath() string

	// Directory that new processes need to chroot into from host's view.
	RootDir() string

	// Directory used by the worker to communicate with container.
	HostDir() string

	// Put the given process into the cgroups of the container
	CGroupEnter(pid string) error

	// PID of a process in the container's namespaces (for joining)
	NSPid() string
}

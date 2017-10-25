package sandbox

import (
	"net/http"
)

/*

Defines the sandbox interface. This interface abstracts all mechanisms
surrounding a given sandbox type (Docker container, cgroup, etc).

*/

import (
	"time"

	"github.com/open-lambda/open-lambda/worker/handler/state"
)

const OLCGroupName = "openlambda"

var CGroupList []string = []string{"blkio", "cpu", "devices", "freezer", "hugetlb", "memory", "perf_event", "systemd"}

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

	ID() string

	RunServer() error

	MemoryCGroupPath() string

	WaitForUnpause(timeout time.Duration) error
}

type ContainerSandbox interface {
	Sandbox

	// Put the passed process into the cgroups of the container
	CGroupEnter(pid string) error

	// PID of a process in the container's namespaces (for joining)
	NSPid() string

	// Directory that new processes need to chroot into (none if docker)
	RootDir() string

	// Directory in the cluster directory to communicate with sandbox
	HostDir() string
}

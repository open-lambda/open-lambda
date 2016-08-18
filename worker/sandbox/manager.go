package sandbox

import (
	docker "github.com/fsouza/go-dockerclient"
	"github.com/phonyphonecall/turnip"
)

type SandboxManager interface {
	Create(name string) (Sandbox, error)
	Pull(name string) error

	// getters
	client() *docker.Client
	createTimer() *turnip.Turnip
	pauseTimer() *turnip.Turnip
	unpauseTimer() *turnip.Turnip
	pullTimer() *turnip.Turnip
	restartTimer() *turnip.Turnip
	inspectTimer() *turnip.Turnip
	startTimer() *turnip.Turnip
	removeTimer() *turnip.Turnip
	logTimer() *turnip.Turnip
}

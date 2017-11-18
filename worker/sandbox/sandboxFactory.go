package sandbox

import (
	"fmt"

	"github.com/open-lambda/open-lambda/worker/config"
)

// SandboxFactory is the common interface for all sandbox creation functions.
type SandboxFactory interface {
	Create(handlerDir, workingDir string) (sandbox Sandbox, err error)
	Cleanup()
}

func InitSandboxFactory(config *config.Config) (sf SandboxFactory, err error) {
	if config.Sandbox == "docker" {
		return NewDockerSBFactory(config)

	} else if config.Sandbox == "olcontainer" {
		return NewOLContainerSBFactory(config)
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", config.Sandbox)
}

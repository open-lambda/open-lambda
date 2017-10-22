package sandbox

import (
	"fmt"

	"github.com/open-lambda/open-lambda/worker/config"
)

const olMntDir = "/tmp/olmnts"

// emptySBInfo contains information necessary for buffers.
type emptySBInfo struct {
	sandbox    Sandbox
	bufDir     string
	handlerDir string
	sandboxDir string
}

// SandboxFactory is the common interface for all sandbox creation functions.
type SandboxFactory interface {
	Create(handlerDir, workingDir string) (sandbox Sandbox, err error)
	Cleanup()
}

func InitSandboxFactory(config *config.Config) (sf SandboxFactory, err error) {
	if config.Sandbox == "docker" {
		delegate, err := NewDockerSBFactory(config)
		if err != nil {
			return nil, err
		}

		if config.Sandbox_buffer == 0 {
			return delegate, nil
		}

		return NewBufferedDockerSBFactory(config, delegate)

	} else if config.Sandbox == "olcontainer" {
		delegate, err := NewOLContainerSBFactory(config)
		if err != nil {
			return nil, err
		}

		if config.Sandbox_buffer == 0 {
			return delegate, nil
		}

		return NewBufferedOLContainerSBFactory(config, delegate)
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", config.Sandbox)
}

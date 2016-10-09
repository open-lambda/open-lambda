package sandbox

import (
	docker "github.com/fsouza/go-dockerclient"
)

type SandboxManager interface {
	Create(name string) (Sandbox, error)
	Pull(name string) error
}

type DockerSandboxManager interface {
	Create(name string) (Sandbox, error)
	Pull(name string) error
	client() *docker.Client
}

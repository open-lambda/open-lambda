package sbmanager

/*

Defines the sandbox manager interfaces. These interfaces abstract all
mechanisms surrounding managing handler code and creating sandboxes for
a given handler code registry.

Sandbox managers are paired with a sandbox interface, which provides
the mechanism for managing an individual sandbox.

*/

import (
	docker "github.com/fsouza/go-dockerclient"
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type SandboxManager interface {
	Create(name string, sandbox_dir string) (sb.Sandbox, error)
	Pull(name string) error
}

type DockerSandboxManager interface {
	Create(name string, sandbox_dir string) (sb.Sandbox, error)
	Pull(name string) error
	client() *docker.Client
}

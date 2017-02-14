package manager

/*

Defines the manager interfaces. These interfaces abstract all mechanisms
surrounding managing handler code and creating sandboxes for a given
handler code registry.

Managers are paired with a sandbox interfaces, which provides functionality
for managing an individual sandbox.

*/

import (
	docker "github.com/fsouza/go-dockerclient"
	sb "github.com/open-lambda/open-lambda/worker/manager/sandbox"
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

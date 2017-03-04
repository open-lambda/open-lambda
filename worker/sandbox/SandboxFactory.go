package sandbox

import (
	"fmt"

	docker "github.com/fsouza/go-dockerclient"
	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/dockerutil"
)

// SandboxFactory is the common interface for all sandbox creation functions.
type SandboxFactory interface {
	Create(handlerDir string, sandboxDir string) (sandbox Sandbox, err error)
}

// DockerSBFactory is a SandboxFactory that creats docker sandboxes.
type DockerSBFactory struct {
	client *docker.Client
	cmd    []string
	labels map[string]string
	env    []string
}

// NewDockerSBFactory creates a DockerSBFactory.
func NewDockerSBFactory(opts *config.Config) (*DockerSBFactory, error) {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	labels := map[string]string{
		dockerutil.DOCKER_LABEL_CLUSTER: opts.Cluster_name,
		dockerutil.DOCKER_LABEL_TYPE:    dockerutil.SANDBOX,
	}
	env := []string{fmt.Sprintf("ol.config=%s", opts.SandboxConfJson())}
	var cmd []string
	if opts.Pool == "" {
		cmd = []string{"/usr/bin/python", "/server.py"}
	} else {
		cmd = []string{"/init"}
	}

	df := &DockerSBFactory{c, cmd, labels, env}
	return df, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (df *DockerSBFactory) Create(handlerDir string, sandboxDir string) (Sandbox, error) {
	volumes := []string{
		fmt.Sprintf("%s:%s", handlerDir, "/handler"),
		fmt.Sprintf("%s:%s", sandboxDir, "/host"),
	}
	container, err := df.client.CreateContainer(
		docker.CreateContainerOptions{
			Config: &docker.Config{
				Image:  dockerutil.BASE_IMAGE,
				Labels: df.labels,
				Env:    df.env,
				Cmd:    df.cmd,
			},
			HostConfig: &docker.HostConfig{
				Binds: volumes,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	sandbox := NewDockerSandbox(sandboxDir, container, df.client)
	return sandbox, nil
}

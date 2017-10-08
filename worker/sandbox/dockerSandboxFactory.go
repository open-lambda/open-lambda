package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/dockerutil"
)

// DockerSBFactory is a SandboxFactory that creats docker sandboxes.
type DockerSBFactory struct {
	client  *docker.Client
	cmd     []string
	labels  map[string]string
	env     []string
	pkgsDir string
}

// NewDockerSBFactory creates a DockerSBFactory.
func NewDockerSBFactory(opts *config.Config) (*DockerSBFactory, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}

	labels := map[string]string{
		dockerutil.DOCKER_LABEL_CLUSTER: opts.Cluster_name,
		dockerutil.DOCKER_LABEL_TYPE:    dockerutil.SANDBOX,
	}
	env := []string{fmt.Sprintf("ol.config=%s", opts.SandboxConfJson())}
	cmd := []string{"/ol-init"}

	df := &DockerSBFactory{client, cmd, labels, env, opts.Pkgs_dir}
	return df, nil
}

// Create creates a docker sandbox from the handler and sandbox directory.
func (df *DockerSBFactory) Create(handlerDir, workingDir, indexHost, indexPort string) (Sandbox, error) {
	id_bytes, err := exec.Command("uuidgen").Output()
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(string(id_bytes[:]))

	// create sandbox directory
	hostDir := path.Join(workingDir, id)
	if err := os.MkdirAll(hostDir, 0777); err != nil {
		return nil, err
	}
	volumes := []string{
		fmt.Sprintf("%s:%s:ro,slave", handlerDir, "/handler"),
		fmt.Sprintf("%s:%s:slave", hostDir, "/host"),
		fmt.Sprintf("%s:%s:ro", df.pkgsDir, "/packages"),
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

	sandbox := NewDockerSandbox(id, hostDir, indexHost, indexPort, container, df.client)
	return sandbox, nil
}

// TODO
func (df *DockerSBFactory) Cleanup() {
	return
}

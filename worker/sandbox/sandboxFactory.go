package sandbox

import (
	"fmt"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/dockerutil"
	"github.com/open-lambda/open-lambda/worker/util"
)

const cacheUnshareFlags = "-iu"
const handlerUnshareFlags = "-ipu"

const cacheCGroupName = "cache"
const handlerCGroupName = "handlers"

const cacheSandboxDir = "/tmp/olcache"
const handlerSandboxDir = "/tmp/olhandlers"

var cacheInitArgs []string = []string{"--cache"}
var handlerInitArgs []string = []string{}

// ContainerFactory is the common interface for creating containers.
type ContainerFactory interface {
	Create(handlerDir, workingDir string) (Container, error)
	Cleanup()
}

func InitCacheContainerFactory(opts *config.Config) (ContainerFactory, error) {
	if opts.Sandbox == "docker" {
		labels := map[string]string{
			dockerutil.DOCKER_LABEL_CLUSTER: opts.Cluster_name,
			dockerutil.DOCKER_LABEL_TYPE:    dockerutil.CACHE,
		}

		return NewDockerContainerFactory(opts, "host", []string{"SYS_ADMIN"}, labels, true)

	} else if opts.Sandbox == "sock" {
		uuid, err := util.UUID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate uuid :: %v", err)
		}
		sandboxDir := fmt.Sprintf("%s-%s", cacheSandboxDir, uuid)

		return NewSOCKContainerFactory(opts, sandboxDir, cacheCGroupName, cacheUnshareFlags, cacheInitArgs)
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", opts.Sandbox)
}

func InitHandlerContainerFactory(opts *config.Config) (ContainerFactory, error) {
	if opts.Sandbox == "docker" {
		labels := map[string]string{
			dockerutil.DOCKER_LABEL_CLUSTER: opts.Cluster_name,
			dockerutil.DOCKER_LABEL_TYPE:    dockerutil.HANDLER,
		}

		return NewDockerContainerFactory(opts, "", nil, labels, false)

	} else if opts.Sandbox == "sock" {
		uuid, err := util.UUID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate uuid :: %v", err)
		}
		sandboxDir := fmt.Sprintf("%s-%s", handlerSandboxDir, uuid)

		return NewSOCKContainerFactory(opts, sandboxDir, handlerCGroupName, handlerUnshareFlags, handlerInitArgs)
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", opts.Sandbox)
}

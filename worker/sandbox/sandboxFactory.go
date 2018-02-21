package sandbox

import (
	"fmt"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/dockerutil"
)

const cacheSandboxDir = "/tmp/olcache"
const handlerSandboxDir = "/tmp/olsbs"

const cacheCGroupName = "cache"
const handlerCGroupName = "handler"

var cacheInitArgs []string = []string{"--cache"}
var handlerInitArgs []string = []string{}

var cacheUnshareFlags []string = []string{"-iu"}
var handlerUnshareFlags []string = []string{"-ipu"}

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
		return NewSOCKContainerFactory(opts, cacheSandboxDir, cacheCGroupName, cacheInitArgs, cacheUnshareFlags)
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
		return NewSOCKContainerFactory(opts, handlerSandboxDir, handlerCGroupName, handlerInitArgs, handlerUnshareFlags)
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", opts.Sandbox)
}

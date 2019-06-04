package sandbox

import (
	"fmt"

	"github.com/open-lambda/open-lambda/ol/config"
	"github.com/open-lambda/open-lambda/ol/sandbox/dockerutil"
	"github.com/open-lambda/open-lambda/ol/util"
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

func InitCacheContainerFactory() (ContainerFactory, error) {
	if config.Conf.Sandbox == "docker" {
		labels := map[string]string{
			dockerutil.DOCKER_LABEL_CLUSTER: config.Conf.Cluster_name,
			dockerutil.DOCKER_LABEL_TYPE:    dockerutil.CACHE,
		}

		return NewDockerContainerFactory("host", []string{"SYS_ADMIN"}, labels, true)

	} else if config.Conf.Sandbox == "sock" {
		uuid, err := util.UUID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate uuid :: %v", err)
		}
		sandboxDir := fmt.Sprintf("%s-%s", cacheSandboxDir, uuid)

		return NewSOCKContainerFactory(sandboxDir, cacheCGroupName, cacheUnshareFlags, cacheInitArgs)
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", config.Conf.Sandbox)
}

func InitHandlerContainerFactory() (ContainerFactory, error) {
	if config.Conf.Sandbox == "docker" {
		labels := map[string]string{
			dockerutil.DOCKER_LABEL_CLUSTER: config.Conf.Cluster_name,
			dockerutil.DOCKER_LABEL_TYPE:    dockerutil.HANDLER,
		}

		return NewDockerContainerFactory("", nil, labels, false)

	} else if config.Conf.Sandbox == "sock" {
		uuid, err := util.UUID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate uuid :: %v", err)
		}
		sandboxDir := fmt.Sprintf("%s-%s", handlerSandboxDir, uuid)

		return NewSOCKContainerFactory(sandboxDir, handlerCGroupName, handlerUnshareFlags, handlerInitArgs)
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", config.Conf.Sandbox)
}

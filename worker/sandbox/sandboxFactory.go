package sandbox

import (
	"fmt"

	"github.com/open-lambda/open-lambda/worker/config"
	"github.com/open-lambda/open-lambda/worker/dockerutil"
)

const cacheSandboxDir = "/tmp/olcache"

var cacheUnshareFlags []string = []string{"-iu"}

const handlerSandboxDir string = "/tmp/olsbs"

var handlerUnshareFlags []string = []string{"-ipu"}

// SandboxFactory is the common interface for all sandbox creation functions.
type SandboxFactory interface {
	Create(handlerDir, workingDir string) (sandbox Sandbox, err error)
	Cleanup()
}

func InitCacheSandboxFactory(opts *config.Config) (sf SandboxFactory, err error) {
	if opts.Sandbox == "docker" {
		labels := map[string]string{
			dockerutil.DOCKER_LABEL_CLUSTER: opts.Cluster_name,
			dockerutil.DOCKER_LABEL_TYPE:    dockerutil.CACHE,
		}

		return NewDockerSBFactory(opts, dockerutil.CACHE_IMAGE, "host", []string{"SYS_ADMIN"}, labels)

	} else if opts.Sandbox == "sock" {
		return NewSOCKSBFactory(opts, opts.SOCK_cache_base, cacheSandboxDir, "cache", cacheUnshareFlags)
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", opts.Sandbox)
}

func InitHandlerSandboxFactory(opts *config.Config) (sf SandboxFactory, err error) {
	if opts.Sandbox == "docker" {
		labels := map[string]string{
			dockerutil.DOCKER_LABEL_CLUSTER: opts.Cluster_name,
			dockerutil.DOCKER_LABEL_TYPE:    dockerutil.HANDLER,
		}

		return NewDockerSBFactory(opts, dockerutil.HANDLER_IMAGE, "", nil, labels)

	} else if opts.Sandbox == "sock" {
		return NewSOCKSBFactory(opts, opts.SOCK_handler_base, handlerSandboxDir, "handler", handlerUnshareFlags)
	}

	return nil, fmt.Errorf("invalid sandbox type: '%s'", opts.Sandbox)
}

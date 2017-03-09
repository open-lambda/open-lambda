package pmanager

import (
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type PoolManager interface {
	ForkEnter(sandbox sb.ContainerSandbox, req_pkgs []string) error
}

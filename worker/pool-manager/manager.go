package pmanager

/*

Defines the manager interfaces. These interfaces abstract all mechanisms
surrounding managing handler code and creating sandboxes for a given
handler code registry.

Managers are paired with a sandbox interfaces, which provides functionality
for managing an individual sandbox.

*/

import (
	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type PoolManager interface {
	ForkEnter(sandbox sb.Sandbox) error
}

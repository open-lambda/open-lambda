package sandbox

import "github.com/open-lambda/open-lambda/worker/handler/state"

type SandboxInfo struct {
	State state.HandlerState
	Port  string
}

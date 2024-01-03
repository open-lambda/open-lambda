package sandbox

import (
	"fmt"
	"strings"
)

// the debugger just watches Sandboxes as they are created, and is
// able to provide a snapshot of the pool at any time
type debugger chan any

func newDebugger(sbPool SandboxPool) debugger {
	var d debugger = make(chan any, 64)
	sbPool.AddListener(func(evType SandboxEventType, sb Sandbox) {
		d <- SandboxEvent{evType, sb}
	})
	go d.Run()
	return d
}

func (d debugger) Run() {
	sandboxes := make(map[string]Sandbox)

	for {
		raw, ok := <-d
		if !ok {
			return
		}

		switch msg := raw.(type) {
		case SandboxEvent:
			switch msg.EvType {
			case EvCreate:
				sandboxes[msg.SB.ID()] = msg.SB
			case EvDestroy:
				delete(sandboxes, msg.SB.ID())
			}
		case chan string:
			var sb strings.Builder

			for _, sandbox := range sandboxes {
				sb.WriteString(fmt.Sprintf("%s--------\n", sandbox.DebugString()))
			}

			msg <- sb.String()
		}
	}
}

func (d debugger) Dump() string {
	ch := make(chan string)
	d <- ch
	return <-ch
}

package sandbox

import (
	"fmt"
	"strings"
)

type debugger chan interface{}

func newDebugger(sbPool SandboxPool) debugger {
	var d debugger = make(chan interface{}, 64)
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
			switch msg.evType {
			case evCreate:
				sandboxes[msg.sb.ID()] = msg.sb
			case evDestroy:
				delete(sandboxes, msg.sb.ID())
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

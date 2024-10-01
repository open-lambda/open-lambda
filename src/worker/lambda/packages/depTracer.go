package packages

import (
	"bufio"
	"encoding/json"
	"os"
)

type DepTracer struct {
	file   *os.File
	writer *bufio.Writer
	events chan map[string]any
	done   chan bool
}

// NewDepTracer creates a new DepTracer instance and initializes it with the given log file path.
func NewDepTracer(logPath string) (*DepTracer, error) {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	t := &DepTracer{
		file:   file,
		writer: bufio.NewWriter(file),
		events: make(chan map[string]any, 128),
		done:   make(chan bool),
	}
	go t.run()

	return t, nil
}

// run processes events and writes them to the log file.
func (t *DepTracer) run() {
	for {
		ev, ok := <-t.events
		if !ok {
			t.writer.Flush()
			t.file.Close()
			t.done <- true
			return
		}

		b, err := json.Marshal(ev)
		if err != nil {
			panic(err)
		}

		t.writer.Write(b)
		t.writer.WriteString("\n")
	}
}

// Cleanup flushes and closes the log file.
func (t *DepTracer) Cleanup() {
	close(t.events)
	<-t.done
}

// TracePackage logs a package event with its dependencies and top-level modules.
func (t *DepTracer) TracePackage(p *Package) {
	t.events <- map[string]any{
		"type": "package",
		"name": p.Name,
		"deps": p.Meta.Deps,
		"top":  p.Meta.TopLevel,
	}
}

// TraceFunction logs a function event with its code directory and direct dependencies.
func (t *DepTracer) TraceFunction(codeDir string, directDeps []string) {
	t.events <- map[string]any{
		"type": "function",
		"name": codeDir,
		"deps": directDeps,
	}
}

// TraceInvocation logs an invocation event with its code directory.
func (t *DepTracer) TraceInvocation(codeDir string) {
	t.events <- map[string]any{
		"type": "invocation",
		"name": codeDir,
	}
}

package policy

import (
	"os"
	"strconv"
	"sync"

	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type ForkServer struct {
	Sandbox  sb.ContainerSandbox
	Pid      string
	SockPath string
	Packages map[string]bool
	Hits     float64
	Parent   *ForkServer
	Children int
	Size     float64
	Mutex    *sync.Mutex
	Dead     bool
}

func (fs *ForkServer) Hit() {
	curr := fs
	for curr != nil {
		curr.Hits += 1.0
		curr = curr.Parent
	}

	return
}

func (fs *ForkServer) Kill() error {
	if fs.Parent == nil {
		panic("attempted to kill the root")
	}

	fs.Dead = true
	pid, err := strconv.Atoi(fs.Pid)
	if err != nil {
		return err
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	proc.Kill()

	go func() {
		fs.Sandbox.Stop()
		fs.Sandbox.Remove()
	}()
	fs.Parent.Children -= 1

	return nil
}

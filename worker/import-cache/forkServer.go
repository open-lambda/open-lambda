package cache

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	sb "github.com/open-lambda/open-lambda/worker/sandbox"
)

type ForkServer struct {
	Sandbox  sb.Container
	Pid      string
	SockPath string
	Imports  map[string]bool
	Hits     float64
	Parent   *ForkServer
	Children int
	Size     float64
	Mutex    *sync.Mutex
	Dead     bool
	Pipe     *os.File
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

	if fs.Parent != nil {
		fs.Parent.Children -= 1
	}

	fs.Sandbox.Unpause()
	fs.Sandbox.Stop()
	fs.Sandbox.Remove()

	return nil
}

func (fs *ForkServer) WaitForEntryInit() error {

	// use StdoutPipe of olcontainer to sync with lambda server
	ready := make(chan bool, 1)
	defer close(ready)
	go func() {
		defer fs.Pipe.Close()

		// wait for "ready"
		buf := make([]byte, 5)
		_, err := fs.Pipe.Read(buf)
		if err != nil {
			log.Printf("Cannot read from stdout of olcontainer: %v\n", err)
		} else if string(buf) != "ready" {
			log.Printf("Expect to read `ready`, but found %v\n", string(buf))
		} else {
			ready <- true
		}
	}()

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	start := time.Now()
	select {
	case <-ready:
		log.Printf("wait for server took %v\n", time.Since(start))
	case <-timeout.C:
		if n, err := fs.Pipe.Write([]byte("timeo")); err != nil {
			return err
		} else if n != 5 {
			return fmt.Errorf("Cannot write `timeo` to pipe\n")
		}
		return fmt.Errorf("Cache entry failed to initialize after 5s")
	}

	return nil
}

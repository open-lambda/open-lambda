package sandbox

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

func waitForServerPipeReady(hostDir string) error {
	// upon success, the goroutine will send nil; else, it will send the error
	ready := make(chan error, 1)

	go func() {
		pipeFile := filepath.Join(hostDir, "server_pipe")
		pipe, err := os.OpenFile(pipeFile, os.O_RDWR, 0777)
		if err != nil {
			log.Printf("Cannot open pipe: %v\n", err)
			return
		}
		defer pipe.Close()

		// wait for "ready"
		buf := make([]byte, 5)
		_, err = pipe.Read(buf)
		if err != nil {
			ready <- fmt.Errorf("Cannot read from stdout of sandbox :: %v\n", err)
		} else if string(buf) != "ready" {
			ready <- fmt.Errorf("Expect to see `ready` but got %s\n", string(buf))
		}
		ready <- nil
	}()

	// TODO: make timeout configurable
	timeout := time.NewTimer(20 * time.Second)
	defer timeout.Stop()

	select {
	case err := <-ready:
		return err
	case <-timeout.C:
		return fmt.Errorf("instance server failed to initialize after 20s")
	}
}

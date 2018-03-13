/*

Utility functions that are used throughout the code.

*/

package util

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"strconv"
	"syscall"
)

// KillPIDStr sends SIGKILL to the process identified by the PID
// passed as a string.
func KillPIDStr(pidStr string) error {
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("bad pid string: %s :: %v", pidStr, err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process with pid: %d :: %v", pid, err)
	}

	err = proc.Signal(syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("failed to send kill signal to process with pid: %d :: %v", pid, err)
	}

	return nil
}

// UUID generates a random UUID according to RFC 4122 (without hyphens).
func UUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x", uuid), nil
}

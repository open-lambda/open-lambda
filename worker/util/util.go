/*

Utility functions that are used throughout the code.

*/

package util

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
)

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

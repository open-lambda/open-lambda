package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// GetLoginUID reads the original login UID from the kernel audit subsystem.
func GetLoginUID() (int, error) {
	data, err := os.ReadFile("/proc/self/loginuid")
	if err != nil {
		return -1, fmt.Errorf("could not read /proc/self/loginuid: %w", err)
	}
	uid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || uid == 4294967295 {
		return -1, fmt.Errorf("loginuid not set; cannot determine the real user")
	}
	return uid, nil
}

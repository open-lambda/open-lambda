package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// See: https://docs.kernel.org/admin-guide/abi-stable.html#proc-pid-loginuid
// GetLoginUID reads the original login UID from the kernel audit subsystem.
//
// /proc/self/loginuid is set by the PAM login module when a user first
// authenticates. Unlike os.Getuid(), which returns the effective UID
// (always 0 under both sudo and su), loginuid always reflects the UID
// of the user who originally logged in.
//
// For example, if user "alice" (UID 1000) runs "sudo ol worker init",
// os.Getuid() returns 0 but loginuid returns 1000. The same is true
// if alice does "su -" and then runs "ol worker init" as root.
func GetLoginUID() (int, error) {
	data, err := os.ReadFile("/proc/self/loginuid")
	if err != nil {
		return -1, fmt.Errorf("could not read /proc/self/loginuid: %w", err)
	}
	uid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	// 2^32 - 1 (unsigned -1) is the default value of loginuid when no login session exists
	if err != nil || uid == 4294967295 {
		return -1, fmt.Errorf("loginuid not set.")
	}
	return uid, nil
}

package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// HostUID returns the *host* uid even if we're uid 0 inside a userns.
func HostUID() int {
	// If we're in a user namespace, uid_map will show the host uid
	if b, err := os.ReadFile("/proc/self/uid_map"); err == nil {
		for _, ln := range strings.Split(string(b), "\n") {
			f := strings.Fields(strings.TrimSpace(ln))
			// first mapping looks like: "0 <host_uid> <size>"
			if len(f) >= 3 && f[0] == "0" {
				if hid, err := strconv.Atoi(f[1]); err == nil {
					// If host uid is 0, we're not in a userns yet (identity mapping)
					if hid != 0 {
						return hid
					}
				}
			}
		}
	}
	if su := os.Getenv("SUDO_UID"); su != "" {
		if hid, err := strconv.Atoi(su); err == nil && hid > 0 {
			return hid
		}
	}
	return os.Getuid()
}

// DelegatedUserCgroupBase returns the systemd user slice path for this user.
func DelegatedUserCgroupBase() (string, error) {
	uid := HostUID()
	base := fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service", uid, uid)
	if st, err := os.Stat(base); err == nil && st.IsDir() {
		return base, nil
	}
	return "", fmt.Errorf("no delegated user slice at uid %d", uid)
}

// GetCgroupDelegationInstructions returns instructions for enabling systemd cgroup delegation.
func GetCgroupDelegationInstructions() string {
	return "Systemd cgroup delegation is not enabled. Please enable it or run with sudo."
}
package common

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CgroupPath() string {
	// Attempt to execute in rootless mode
	cmd := exec.Command("systemctl", "show", "--user", "--property=ControlGroup")
	output, err := cmd.Output()
	if err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(output)), "=", 2)
		if len(parts) == 2 {
			path := filepath.Join("/sys/fs/cgroup", parts[1])
			if st, err := os.Stat(path); err == nil && st.IsDir() {
				return path
			}
		}
	}

	// Use default path if run as root
	if os.Getuid() == 0 {
		return "/sys/fs/cgroup"
	}

	panic(fmt.Errorf("systemd user cgroup delegation not available - cannot run rootless"))
}

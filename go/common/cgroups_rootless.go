package common

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CgroupPath() string {
	cmd := exec.Command("systemctl", "show", "--user", "--property=ControlGroup")
	output, err := cmd.Output()
	if err != nil {
		panic(fmt.Errorf("systemd is required for rootless operation: %w", err))
	}

	parts := strings.SplitN(strings.TrimSpace(string(output)), "=", 2)
	if len(parts) != 2 {
		panic(fmt.Errorf("unexpected systemctl output format: %q", string(output)))
	}

	path := filepath.Join("/sys/fs/cgroup", parts[1])
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		panic(fmt.Errorf("systemd user cgroup not found - delegation required: %s", path))
	}

	return path
}

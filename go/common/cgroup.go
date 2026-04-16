package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CgroupRoot() (string, error) {
	if Conf != nil && Conf.Cgroup_root != "" {
		return Conf.Cgroup_root, nil
	}
	data, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", fmt.Errorf("read /proc/self/cgroup: %w", err)
	}
	line := strings.TrimSpace(string(data))
	parts := strings.SplitN(line, "::", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected /proc/self/cgroup format: %q", line)
	}
	return filepath.Join("/sys/fs/cgroup", parts[1]), nil
}

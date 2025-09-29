package worker

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// hostUID returns the *host* uid even if we're uid 0 inside a userns.
func hostUID() int {
	if b, err := os.ReadFile("/proc/self/uid_map"); err == nil {
		for _, ln := range strings.Split(string(b), "\n") {
			f := strings.Fields(strings.TrimSpace(ln))
			// first mapping looks like: "0 <host_uid> <size>"
			if len(f) >= 2 && f[0] == "0" {
				if hid, err := strconv.Atoi(f[1]); err == nil {
					return hid
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

// delegatedUserCgroupBase returns the systemd user slice path for this user.
func delegatedUserCgroupBase() (string, error) {
	uid := hostUID()
	base := fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/user.slice", uid, uid)
	if st, err := os.Stat(base); err == nil && st.IsDir() {
		return base, nil
	}
	return "", fmt.Errorf("no delegated user slice at uid %d", uid)
}

// ResolveCgroupPoolPath picks where to create the pool.
// Returns (path, disableCgroups). If disableCgroups is true, skip cgroup writes.
func ResolveCgroupPoolPath(clusterName string) (string, bool) {
	// try systemd user slice (rootless-friendly)
	if base, err := delegatedUserCgroupBase(); err == nil {
		p := filepath.Join(base, clusterName+"-sandboxes.slice")
		if err := os.MkdirAll(p, 0o755); err == nil {
			_ = enableControllersBestEffort(base, []string{"+cpu", "+memory", "+pids"})
			return p, false
		}
		log.Printf("WARN: cannot create %s (%v); running without cgroups", p, err)
		return "", true
	}

	// fallback for rootful/legacy
	p := filepath.Join("/sys/fs/cgroup", clusterName+"-sandboxes")
	if err := os.MkdirAll(p, 0o755); err == nil {
		return p, false
	}
	log.Printf("WARN: cannot create %s; running without cgroups", p)
	return "", true
}

// enableControllersBestEffort tries to enable controllers under the base.
// Failures are fine (depends on systemd delegation).
func enableControllersBestEffort(base string, plus []string) error {
	sc := filepath.Join(base, "cgroup.subtree_control")
	f, err := os.OpenFile(sc, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	for _, t := range plus {
		_, _ = f.WriteString(t + "\n")
	}
	return nil
}

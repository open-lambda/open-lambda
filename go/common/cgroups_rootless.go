package common

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

func HostUID() int {
	file, err:= os.ReadFile("/proc/self/uid_map");
	if err != nil {
		panic(fmt.Errorf("Read: %s", err))
	}

	for ln := range strings.SplitSeq(string(file), "\n") {
		f := strings.Fields(strings.TrimSpace(ln))
		// "0 <host_uid> <size>"
		if len(f) >= 3 && f[0] == "0" {
			if hid, err := strconv.Atoi(f[1]); err == nil {
				// If host uid is 0, we're not in a userns yet
				if hid != 0 {
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

// Returns the systemd user slice path for this user.
func CgroupPath() string {
	uid := HostUID()
	base := fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service", uid, uid)
	if st, err := os.Stat(base); err != nil || !st.IsDir(){
		panic(fmt.Errorf("%s: no delegated user slice at uid %d", err, uid))
	}
	return base
}

package cgroups

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/go/common"
)

// if there are fewer than CGROUP_RESERVE available, more will be created.
// If there are more than 2*CGROUP_RESERVE available, they'll be released.
const CGROUP_RESERVE = 16

type CgroupPool struct {
	Name     string
	ready    chan *CgroupImpl
	recycled chan *CgroupImpl
	quit     chan chan bool
	nextID   int
}

// / NOTE (rootless): helpers used only for delegated user-slice resolution.
func hostUIDForCgroups() int {
	if b, err := os.ReadFile("/proc/self/uid_map"); err == nil {
		for _, ln := range strings.Split(string(b), "\n") {
			ln = strings.TrimSpace(ln)
			if ln == "" {
				continue
			}
			fs := strings.Fields(ln)
			// first mapping: "0 <host_uid> <size>"
			if len(fs) >= 2 && fs[0] == "0" {
				if hid, err := strconv.Atoi(fs[1]); err == nil && hid >= 0 {
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

// NOTE (rootless): prefer systemd user slice when present for cgroup pool.
func delegatedUserCgroupBase() (string, error) {
	uid := hostUIDForCgroups()
	p := fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/user.slice", uid, uid)
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p, nil
	}
	return "", fmt.Errorf("delegated user cgroup base not found for uid %d", uid)
}

// NOTE (rootless): best-effort guard for controller files.
func writeOK(p string) bool {
	st, err := os.Stat(p)
	if err != nil || !st.Mode().IsRegular() {
		return false
	}
	f, err := os.OpenFile(p, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	_ = f.Close()
	return true
}

// NewCgroupPool creates a new CgroupPool with the specified name.
func NewCgroupPool(name string) (*CgroupPool, error) {
	pool := &CgroupPool{
		Name:     path.Base(path.Dir(common.Conf.Worker_dir)) + "-" + name,
		ready:    make(chan *CgroupImpl, CGROUP_RESERVE),
		recycled: make(chan *CgroupImpl, CGROUP_RESERVE),
		quit:     make(chan chan bool),
		nextID:   0,
	}

	// create (or ensure) the pool directory
	groupPath := pool.GroupPath()
	pool.printf("using cgroup base: %s", groupPath)
	if err := os.MkdirAll(groupPath, 0o700); err != nil {
		return nil, fmt.Errorf("MkdirAll %s: %w", groupPath, err)
	}

	// Best-effort: make controllers available to child groups.
	// Not all Ubuntu/systemd setups delegate +cpu/+memory/+pids to user slices.
	// We ignore failures here and let later code skip writes if delegation is missing.
	rpath := fmt.Sprintf("%s/cgroup.subtree_control", groupPath)
	if f, err := os.OpenFile(rpath, os.O_WRONLY|os.O_APPEND, 0); err == nil {
		_, _ = f.WriteString("+pids +io +memory +cpu\n")
		_ = f.Close()
	} else {
		pool.printf("WARN: could not write %s (%v); continuing without delegating controllers", rpath, err)
	}

	go pool.cgTask()
	return pool, nil
}

// NewCgroup creates a new CGroup in the pool
func (pool *CgroupPool) NewCgroup() Cgroup {
	for {
		pool.nextID++

		cg := &CgroupImpl{
			name: fmt.Sprintf("cg-%d", pool.nextID),
			pool: pool,
		}

		groupPath := cg.GroupPath()
		if err := os.Mkdir(groupPath, 0700); err != nil {
			// If a previous run left cg-N behind, try the next N
			if os.IsExist(err) {
				continue
			}
			panic(fmt.Errorf("Mkdir %s: %s", groupPath, err))
		}

		cg.printf("created")
		return cg
	}
}

// add ID to each log message so we know which logs correspond to
// which containers
func (pool *CgroupPool) printf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	slog.Info(fmt.Sprintf("%s [CGROUP POOL %s]", strings.TrimRight(msg, "\n"), pool.Name))
}

func (pool *CgroupPool) cgTask() {
	// we'll be sent this as part of the quit request
	var done chan bool

	// loop until we get the quit message
	pool.printf("start creating/serving CGs")
Loop:
	for {
		var cg *CgroupImpl

		// get a new or recycled cgroup.  Settings may be initialized
		// in one of three places, the first two of which are here:
		//
		// 1. upon fresh creation (things that never change, such as max procs)
		// 2. after it's been recycled (we need to clean things up that change during use)
		// 3. some things (e.g., memory limits) need to be done in either case, and may
		//    depend on the needs of the Sandbox; this happens in pool.GetCg (which is
		//    fed by this function)
		select {
		case cg = <-pool.recycled:
			// restore cgroup to clean state
			// FIXME not possible in CG2?
			// cg.WriteInt("memory.failcnt", 0)
			cg.Unpause()
		default:
			t := common.T0("fresh-cgroup")
			cg = pool.NewCgroup().(*CgroupImpl)
			// Only attempt controller writes if the files are present & writable
			pidsPath := path.Join(cg.GroupPath(), "pids.max")
			swapPath := path.Join(cg.GroupPath(), "memory.swap.max")

			if writeOK(pidsPath) {
				cg.WriteInt("pids.max", int64(common.Conf.Limits.Procs))
			} else {
				cg.printf("WARN: skipping write pids.max (no delegation/permission)")
			}
			if writeOK(swapPath) {
				cg.WriteInt("memory.swap.max", int64(common.Conf.Limits.Swappiness))
			} else {
				cg.printf("WARN: skipping write memory.swap.max (no delegation/permission)")
			}
			t.T1()
		}

		// add cgroup to ready queue
		select {
		case pool.ready <- cg:
		case done = <-pool.quit:
			pool.printf("received shutdown request")
			cg.Destroy()
			break Loop
		}
	}

	// empty queues, freeing all cgroups
	pool.printf("empty queues and release CGs")
Empty:
	for {
		select {
		case cg := <-pool.ready:
			cg.Destroy()
		case cg := <-pool.recycled:
			cg.Destroy()
		default:
			break Empty
		}
	}

	done <- true
}

// Destroy this entire cgroup pool
func (pool *CgroupPool) Destroy() {
	// signal cgTask, then wait for it to finish
	ch := make(chan bool)
	pool.quit <- ch
	<-ch

	// Destroy cgroup for this entire pool
	gpath := pool.GroupPath()
	pool.printf("Destroying cgroup pool with path \"%s\"", gpath)
	for i := 100; i >= 0; i-- {
		if err := syscall.Rmdir(gpath); err != nil {
			if i == 0 {
				panic(fmt.Errorf("Rmdir %s: %s", gpath, err))
			}

			pool.printf("cgroup pool Rmdir failed, trying again in 5ms")
			time.Sleep(5 * time.Millisecond)
		} else {
			break
		}
	}
}

// GetCg retrieves a cgroup from the pool, setting its memory limit and CPU percentage.
func (pool *CgroupPool) GetCg(memLimitMB int, moveMemCharge bool, cpuPercent int) Cgroup {
	cg := <-pool.ready
	// Guard writes so missing delegation doesn't kill the worker
	memMax := path.Join(cg.GroupPath(), "memory.max")
	cpuWeight := path.Join(cg.GroupPath(), "cpu.weight")

	if writeOK(memMax) {
		cg.SetMemLimitMB(memLimitMB)
	} else {
		cg.printf("WARN: skipping memory.max (no delegation/permission)")
	}
	if writeOK(cpuWeight) {
		cg.SetCPUPercent(cpuPercent)
	} else {
		cg.printf("WARN: skipping cpu.weight (no delegation/permission)")
	}

	// FIXME not supported in CG2?
	var _ = moveMemCharge
	return cg
}

// GroupPath returns the path to the Cgroup pool for OpenLambda
func (pool *CgroupPool) GroupPath() string {
	if base, err := delegatedUserCgroupBase(); err == nil {
		name := pool.Name
		if !strings.HasSuffix(name, ".slice") {
			name += ".slice"
		}
		return path.Join(base, name)
	}
	return fmt.Sprintf("/sys/fs/cgroup/%s", pool.Name)
}

package cgroups

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/open-lambda/open-lambda/go/common"
)

// if there are fewer than CGROUP_RESERVE available, more will be created.
// If there are more than 2*CGROUP_RESERVE available, they'll be released.
const CGROUP_RESERVE = 16

type CgroupPool struct {
	Name       string
	parentPath string
	ready      chan *CgroupImpl
	recycled   chan *CgroupImpl
	quit       chan chan bool
	nextID     int
}

// NewCgroupPool creates a new CgroupPool with the specified name.
func NewCgroupPool(name, parentPath string) (*CgroupPool, error) {
	pool := &CgroupPool{
		Name:       path.Base(path.Dir(common.Conf.Worker_dir)) + "-" + name,
		parentPath: parentPath,
		ready:      make(chan *CgroupImpl, CGROUP_RESERVE),
		recycled:   make(chan *CgroupImpl, CGROUP_RESERVE),
		quit:       make(chan chan bool),
		nextID:     0,
	}

	pool.printf("using parent cgroup %s", parentPath)

	// create cgroup pool parent - no-op if already exists
	if err := os.MkdirAll(parentPath, 0700); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", parentPath, err)
	}

	// enumerate controllers delegated to the parent
	ctrlData, err := os.ReadFile(filepath.Join(parentPath, "cgroup.controllers"))
	if err != nil {
		return nil, fmt.Errorf("read %s/cgroup.controllers: %w", parentPath, err)
	}
	available := strings.Fields(string(ctrlData))

	// require cpu, memory, pids
	for _, req := range []string{"cpu", "memory", "pids"} {
		if !slices.Contains(available, req) {
			return nil, fmt.Errorf(
				"cannot delegate required cgroup controller %q to %s (available: %v); ",
				req, parentPath, available,
			)
		}
	}

	// move self pid to worker cgroup before enabling subtree_control
	workerPath := filepath.Join(parentPath, "worker")
	if err := os.Mkdir(workerPath, 0700); err != nil && !os.IsExist(err) {
		return nil, fmt.Errorf("mkdir %s: %w", workerPath, err)
	}
	// migrate every pid in parent to worker/ - --detach leaves >1 in the scope
	parentProcs, err := os.ReadFile(filepath.Join(parentPath, "cgroup.procs"))
	if err != nil {
		return nil, fmt.Errorf("read %s/cgroup.procs: %w", parentPath, err)
	}
	workerProcsPath := filepath.Join(workerPath, "cgroup.procs")
	for _, pidStr := range strings.Fields(string(parentProcs)) {
		// pid exited between read and write - ignore
		if err := os.WriteFile(workerProcsPath, []byte(pidStr), 0); err != nil && !errors.Is(err, syscall.ESRCH) {
			return nil, fmt.Errorf("move pid %s into %s: %w", pidStr, workerPath, err)
		}
	}

	// enable every available controller on parent's subtree_control
	var enable strings.Builder
	for i, c := range available {
		if i > 0 {
			enable.WriteByte(' ')
		}
		enable.WriteByte('+')
		enable.WriteString(c)
	}
	subPath := filepath.Join(parentPath, "cgroup.subtree_control")
	if err := os.WriteFile(subPath, []byte(enable.String()), 0); err != nil {
		return nil, fmt.Errorf("enable controllers at %s: %w", subPath, err)
	}

	go pool.cgTask()
	return pool, nil
}

// NewCgroup creates a new CGroup in the pool
func (pool *CgroupPool) NewCgroup() Cgroup {
	pool.nextID++

	cg := &CgroupImpl{
		name: fmt.Sprintf("cg-%d", pool.nextID),
		pool: pool,
	}

	groupPath := cg.GroupPath()
	if err := syscall.Mkdir(groupPath, 0700); err != nil {
		panic(fmt.Errorf("Mkdir %s: %s", groupPath, err))
	}

	cg.printf("created")
	return cg
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
			cg.WriteInt("pids.max", int64(common.Conf.Limits.Procs))
			cg.WriteInt("memory.swap.max", int64(common.Conf.Limits.Swappiness))
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
	pool.printf("cgroup pool drained")
}

// GetCg retrieves a cgroup from the pool, setting its memory limit and CPU percentage.
func (pool *CgroupPool) GetCg(memLimitMB int, moveMemCharge bool, cpuPercent int) Cgroup {
	cg := <-pool.ready
	cg.SetMemLimitMB(memLimitMB)
	cg.SetCPUPercent(cpuPercent)

	// FIXME not supported in CG2?
	var _ = moveMemCharge

	/*
		if moveMemCharge {
			cg.WriteInt("memory.move_charge_at_immigrate", 1)
		} else {
			cg.WriteInt("memory.move_charge_at_immigrate", 0)
		}*/

	return cg
}

// GroupPath returns the path to the Cgroup pool for OpenLambda
func (pool *CgroupPool) GroupPath() string {
	return pool.parentPath
}

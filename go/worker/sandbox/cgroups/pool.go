package cgroups

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/open-lambda/open-lambda/go/common"
)

// if there are fewer than CGROUP_RESERVE available, more will be created.
// If there are more than 2*CGROUP_RESERVE available, they'll be released.
const CGROUP_RESERVE = 16

type CgroupPool struct {
	Name     string
	poolPath string
	ready    chan *CgroupImpl
	recycled chan *CgroupImpl
	quit     chan chan bool
	nextID   int
}

// InitPoolRoot creates the cgroup pool root directory and enables controllers.
func InitPoolRoot(poolPath string) error {

	if err := os.MkdirAll(poolPath, 0700); err != nil {
		return fmt.Errorf("failed to create cgroup pool root %s: %w", poolPath, err)
	}

	ctrlPath := filepath.Join(poolPath, "cgroup.subtree_control")
	if err := os.WriteFile(ctrlPath, []byte("+pids +io +memory +cpu"), os.ModeAppend); err != nil {
		return fmt.Errorf("failed to enable controllers at %s: %w", ctrlPath, err)
	}

	uid, err := common.GetLoginUID()
	if err != nil {
		return fmt.Errorf("failed to determine real user: %w", err)
	}
	if err := os.Chown(poolPath, uid, uid); err != nil {
		return fmt.Errorf("failed to chown cgroup pool root: %w", err)
	}

	fmt.Printf("\tCreated cgroup pool root at %s\n", poolPath)
	return nil
}

func NewCgroupPool(name string, poolPath string) (*CgroupPool, error) {
	pool := &CgroupPool{
		Name:     name,
		poolPath: poolPath,
		ready:    make(chan *CgroupImpl, CGROUP_RESERVE),
		recycled: make(chan *CgroupImpl, CGROUP_RESERVE),
		quit:     make(chan chan bool),
		nextID:   0,
	}

	if st, err := os.Stat(poolPath); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("cgroup pool root %s does not exist.", poolPath)
	}

	pool.printf("reusing pool root %s", poolPath)
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

// Destroy drains all child cgroups but preserves the pool root.
func (pool *CgroupPool) Destroy() {
	// signal cgTask, then wait for it to finish
	ch := make(chan bool)
	pool.quit <- ch
	<-ch

	pool.printf("destroyed all child cgroups, pool root preserved")
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

// GroupPath returns the path to the cgroup pool root directory.
func (pool *CgroupPool) GroupPath() string {
	return pool.poolPath
}

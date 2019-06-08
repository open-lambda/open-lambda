package sandbox

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var cgroupList []string = []string{
	"blkio", "cpu", "devices", "freezer", "hugetlb",
	"memory", "perf_event", "systemd"}

// if there are fewer than CGROUP_RESERVE available, more will be created.
// If there are more than 2*CGROUP_RESERVE available, they'll be released.
const CGROUP_RESERVE = 16

type Cgroup struct {
	Name string
	pool *CgroupPool
}

type CgroupPool struct {
	Name     string
	ready    chan *Cgroup
	recycled chan *Cgroup
	quit     chan bool
	nextId   int
}

func NewCgroupPool(name string) *CgroupPool {
	pool := &CgroupPool{
		Name:     name,
		ready:    make(chan *Cgroup, CGROUP_RESERVE),
		recycled: make(chan *Cgroup, CGROUP_RESERVE),
		quit:     make(chan bool),
		nextId:   0,
	}

	go pool.cgTask()

	return pool
}

func (pool *CgroupPool) cgTask() {
	for {
		var cg *Cgroup

		// get a new or recycled cgroup
		select {
		case cg = <-pool.recycled:
			// restore cgroup to clean state
			cg.Unpause()
		default:
			pool.nextId += 1
			cg = &Cgroup{
				Name: fmt.Sprintf("cg-%d", pool.nextId),
				pool: pool,
			}
			cg.Init()
		}

		// add cgroup to ready queue
		select {
		case pool.ready <- cg:
		case <-pool.quit:
			pool.destroy()
			return
		}
	}
}

func (pool *CgroupPool) Destroy() {
	pool.quit <- true
}

func (pool *CgroupPool) destroy() {
	// empty queues, freeing all cgroups
	for {
		select {
		case cg := <-pool.ready:
			cg.destroy()
		case cg := <-pool.recycled:
			cg.destroy()
		default:
			break
		}
	}

	// delete cgroup categories
	for _, resource := range cgroupList {
		if err := os.RemoveAll(pool.Path(resource)); err != nil {
			panic(err)
		}
	}
}

func (pool *CgroupPool) GetCg() *Cgroup {
	return <-pool.ready
}

func (pool *CgroupPool) Path(resource string) string {
	return fmt.Sprintf("/sys/fs/cgroup/%s/%s", resource, pool.Name)
}

func (cg *Cgroup) Path(resource, filename string) string {
	if filename == "" {
		return fmt.Sprintf("%s/%s", cg.pool.Path(resource), cg.Name)
	}
	return fmt.Sprintf("%s/%s/%s", cg.pool.Path(resource), cg.Name, filename)
}

func (cg *Cgroup) AddPid(pid string) error {
	// put process into each cgroup
	for _, resource := range cgroupList {
		err := ioutil.WriteFile(cg.Path(resource, "tasks"), []byte(pid), os.ModeAppend)
		if err != nil {
			return err
		}
	}

	return nil
}

func (cg *Cgroup) setFreezeState(state string) error {
	freezerPath := cg.Path("freezer", "freezer.state")
	err := ioutil.WriteFile(freezerPath, []byte(state), os.ModeAppend)
	if err != nil {
		return err
	}

	timeout := 5 * time.Second

	start := time.Now()
	for time.Since(start) < timeout {
		freezerState, err := ioutil.ReadFile(freezerPath)
		if err != nil {
			return fmt.Errorf("failed to check self_freezing state :: %v", err)
		}

		if strings.TrimSpace(string(freezerState[:])) == state {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}

	return fmt.Errorf("sock didn't pause/unpause after %v", timeout)
}

func (cg *Cgroup) Pause() error {
	return cg.setFreezeState("FROZEN")
}

func (cg *Cgroup) Unpause() error {
	return cg.setFreezeState("THAWED")
}

// recommended cleanup protocol:
// 1. Pause (otherwise new procs may be spawned while KillAllProcs runs)
// 2. KillAllProcs
// 3. Unpause (otherwise kill cannot finish)
func (cg *Cgroup) KillAllProcs() error {
	procsPath := cg.Path("memory", "cgroup.procs")
	pids, err := ioutil.ReadFile(procsPath)
	if err != nil {
		return err
	}

	// TODO: this is racy: what if processes are being created as we're killing them?
	// can we freeze the cgroup, then kill them?
	for _, pidStr := range strings.Split(strings.TrimSpace(string(pids[:])), "\n") {
		if pidStr == "" {
			break
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			return fmt.Errorf("bad pid string: %s :: %v", pidStr, err)
		}

		proc, err := os.FindProcess(pid)
		if err != nil {
			fmt.Errorf("failed to find process with pid: %d :: %v", pid, err)
		}

		// forced termination (not trappable)
		err = proc.Signal(syscall.SIGKILL)
		if err != nil {
			fmt.Errorf("failed to send kill signal to process with pid: %d :: %v", pid, err)
		}
	}

	return nil
}

func (cg *Cgroup) Init() {
	for _, resource := range cgroupList {
		if err := os.MkdirAll(cg.Path(resource, ""), 0700); err != nil {
			panic(err)
		}
	}
}

func (cg *Cgroup) Release() {
	// TODO: assert that there are no tasks remaining

	// if there's room in the recycled channel, add it there.
	// Otherwise, just delete it.
	select {
	case cg.pool.recycled <- cg:
	default:
		cg.destroy()
	}
}

func (cg *Cgroup) destroy() {
	for _, resource := range cgroupList {
		if err := os.RemoveAll(cg.Path(resource, "")); err != nil {
			panic(err)
		}
	}
}

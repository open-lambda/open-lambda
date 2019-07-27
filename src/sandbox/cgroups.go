package sandbox

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/ol/common"
)

var cgroupList []string = []string{
	"blkio", "cpu", "devices", "freezer", "hugetlb",
	"memory", "perf_event", "systemd", "pids"}

// if there are fewer than CGROUP_RESERVE available, more will be created.
// If there are more than 2*CGROUP_RESERVE available, they'll be released.
const CGROUP_RESERVE = 16

type Cgroup struct {
	Name       string
	pool       *CgroupPool
	memLimitMB int
}

type CgroupPool struct {
	Name     string
	ready    chan *Cgroup
	recycled chan *Cgroup
	quit     chan chan bool
	nextId   int
}

func NewCgroupPool(name string) (*CgroupPool, error) {
	pool := &CgroupPool{
		Name:     path.Base(path.Dir(common.Conf.Worker_dir)) + "-" + name,
		ready:    make(chan *Cgroup, CGROUP_RESERVE),
		recycled: make(chan *Cgroup, CGROUP_RESERVE),
		quit:     make(chan chan bool),
		nextId:   0,
	}

	// create cgroup categories
	for _, resource := range cgroupList {
		path := pool.Path(resource)
		pool.printf("create %s", path)
		if err := syscall.Mkdir(path, 0700); err != nil {
			return nil, fmt.Errorf("Mkdir %s: %s", path, err)
		}
	}

	go pool.cgTask()
	return pool, nil
}

func (pool *CgroupPool) NewCgroup() *Cgroup {
	pool.nextId += 1
	cg := &Cgroup{
		Name: fmt.Sprintf("cg-%d", pool.nextId),
		pool: pool,
	}

	for _, resource := range cgroupList {
		path := cg.Path(resource, "")
		if err := syscall.Mkdir(path, 0700); err != nil {
			panic(fmt.Errorf("Mkdir %s: %s", path, err))
		}
	}

	cg.printf("created")
	return cg
}

func (cg *Cgroup) printf(format string, args ...interface{}) {
	if common.Conf.Trace.Cgroups {
		msg := fmt.Sprintf(format, args...)
		log.Printf("%s [CGROUP %s: %s]", strings.TrimRight(msg, "\n"), cg.pool.Name, cg.Name)
	}
}

func (cg *Cgroup) Release() {
	pids, err := cg.GetPIDs()
	if err != nil {
		panic(err)
	} else if len(pids) != 0 {
		panic(fmt.Errorf("Cannot release cgroup that contains processes: %v", pids))
	}

	// if there's room in the recycled channel, add it there.
	// Otherwise, just delete it.
	if common.Conf.Features.Reuse_cgroups {
		select {
		case cg.pool.recycled <- cg:
			cg.printf("release and recycle")
			return
		default:
		}
	}

	cg.printf("release and destroy")
	cg.destroy()
}

func (cg *Cgroup) destroy() {
	for _, resource := range cgroupList {
		path := cg.Path(resource, "")
		if err := syscall.Rmdir(path); err != nil {
			panic(fmt.Errorf("Rmdir %s: %s", path, err))
		}
	}
}

// add ID to each log message so we know which logs correspond to
// which containers
func (pool *CgroupPool) printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("%s [CGROUP POOL %s]", strings.TrimRight(msg, "\n"), pool.Name)
}

func (pool *CgroupPool) cgTask() {
	// we'll be sent this as part of the quit request
	var done chan bool

	// loop until we get the quit message
	pool.printf("start creating/serving CGs")
Loop:
	for {
		var cg *Cgroup

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
			cg.WriteInt("memory", "memory.failcnt", 0)
			cg.Unpause()
		default:
			t := common.T0("fresh-cgroup")
			cg = pool.NewCgroup()
			cg.WriteInt("pids", "pids.max", int64(common.Conf.Limits.Procs))
			cg.WriteInt("memory", "memory.swappiness", int64(common.Conf.Limits.Swappiness))
			t.T1()
		}

		// add cgroup to ready queue
		select {
		case pool.ready <- cg:
		case done = <-pool.quit:
			pool.printf("received shutdown request")
			cg.destroy()
			break Loop
		}
	}

	// empty queues, freeing all cgroups
	pool.printf("empty queues and release CGs")
Empty:
	for {
		select {
		case cg := <-pool.ready:
			cg.destroy()
		case cg := <-pool.recycled:
			cg.destroy()
		default:
			break Empty
		}
	}

	done <- true
}

func (pool *CgroupPool) Destroy() {
	// signal cgTask, then wait for it to finish
	ch := make(chan bool)
	pool.quit <- ch
	<-ch

	// delete cgroup categories
	for _, resource := range cgroupList {
		path := pool.Path(resource)
		pool.printf("remove %s", path)
		if err := syscall.Rmdir(path); err != nil {
			panic(fmt.Errorf("Rmdir %s: %s", path, err))
		}
	}
}

func (pool *CgroupPool) GetCg(memLimitMB int, moveMemCharge bool) *Cgroup {
	cg := <-pool.ready
	cg.setMemLimitMB(memLimitMB)
	if moveMemCharge {
		cg.WriteInt("memory", "memory.move_charge_at_immigrate", 1)
	} else {
		cg.WriteInt("memory", "memory.move_charge_at_immigrate", 0)
	}
	return cg
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

func (cg *Cgroup) TryWriteInt(resource, filename string, val int64) error {
	return ioutil.WriteFile(cg.Path(resource, filename), []byte(fmt.Sprintf("%d", val)), os.ModeAppend)
}

func (cg *Cgroup) WriteInt(resource, filename string, val int64) {
	if err := cg.TryWriteInt(resource, filename, val); err != nil {
		panic(fmt.Sprintf("Error writing %v to %s of %s: %v", val, filename, resource, err))
	}
}

func (cg *Cgroup) TryReadInt(resource, filename string) (int64, error) {
	raw, err := ioutil.ReadFile(cg.Path(resource, filename))
	if err != nil {
		return 0, err
	}
	val, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (cg *Cgroup) ReadInt(resource, filename string) int64 {
	if val, err := cg.TryReadInt(resource, filename); err != nil {
		panic(err)
	} else {
		return val
	}
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
	for {
		freezerState, err := ioutil.ReadFile(freezerPath)
		if err != nil {
			return fmt.Errorf("failed to check self_freezing state :: %v", err)
		}

		if strings.TrimSpace(string(freezerState)) == state {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("cgroup stuck on %s after %v (should be %s)", freezerState, timeout, state)
		}

		time.Sleep(1 * time.Millisecond)
	}
}

// get mem usage in MB
func (cg *Cgroup) getMemUsageMB() int {
	usage := cg.ReadInt("memory", "memory.usage_in_bytes")

	// round up to nearest MB
	mb := int64(1024 * 1024)
	return int((usage + mb - 1) / mb)
}

// get mem limit in MB
func (cg *Cgroup) getMemLimitMB() int {
	return cg.memLimitMB
}

// set mem limit in MB
func (cg *Cgroup) setMemLimitMB(mb int) {
	if mb == cg.memLimitMB {
		return
	}

	limitPath := cg.Path("memory", "memory.limit_in_bytes")
	bytes := int64(mb) * 1024 * 1024
	cg.WriteInt("memory", "memory.limit_in_bytes", bytes)

	// cgroup v1 documentation recommends reading back limit after
	// writing, because it is only a suggestion (e.g., may get
	// rounded to page size).
	//
	// we don't have a great way of dealing with this now, so
	// we'll just panic if it is not within some tolerance
	limitRaw, err := ioutil.ReadFile(limitPath)
	if err != nil {
		panic(err)
	}
	limit, err := strconv.ParseInt(strings.TrimSpace(string(limitRaw)), 10, 64)
	if err != nil {
		panic(err)
	}

	diff := limit - bytes
	if diff < -1024*1024 || diff > 1024*1024 {
		panic(fmt.Errorf("tried to set mem limit to %d, but result (%d) was not within 1MB tolerance",
			bytes, limit))
	}

	cg.memLimitMB = mb
}

func (cg *Cgroup) Pause() error {
	return cg.setFreezeState("FROZEN")
}

func (cg *Cgroup) Unpause() error {
	return cg.setFreezeState("THAWED")
}

func (cg *Cgroup) GetPIDs() ([]string, error) {
	// we could use any cgroup resource type, as they should all
	// have the same procs
	procsPath := cg.Path("freezer", "tasks")
	pids, err := ioutil.ReadFile(procsPath)
	if err != nil {
		return nil, err
	}

	pidStr := strings.TrimSpace(string(pids))
	if len(pidStr) == 0 {
		return []string{}, nil
	}

	return strings.Split(pidStr, "\n"), nil
}

// CG most be paused beforehand
func (cg *Cgroup) KillAllProcs() []string {
	pids, err := cg.GetPIDs()
	if err != nil {
		panic(err)
	}

	for _, pidStr := range pids {
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			panic(fmt.Errorf("bad pid string: %s :: %v", pidStr, err))
		}

		proc, err := os.FindProcess(pid)
		if err != nil {
			panic(fmt.Errorf("failed to find process with pid: %d :: %v", pid, err))
		}

		// forced termination (not trappable)
		err = proc.Signal(syscall.SIGKILL)
		if err != nil {
			panic(fmt.Errorf("failed to send kill signal to process with pid: %d :: %v", pid, err))
		}
	}

	if err := cg.Unpause(); err != nil {
		panic(err)
	}

Loop:
	for i := 0; ; i++ {
		pids, err := cg.GetPIDs()
		if err != nil {
			panic(err)
		} else if len(pids) == 0 {
			break Loop
		} else if i%1000 == 0 {
			cg.pool.printf("waiting for %d procs in %s to die", len(pids), cg.Name)
		}
		time.Sleep(1 * time.Millisecond)
	}

	return pids
}

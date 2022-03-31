package sandbox

import (
	"bufio"
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
	nextID   int
}

func NewCgroupPool(name string) (*CgroupPool, error) {
	pool := &CgroupPool{
		Name:     path.Base(path.Dir(common.Conf.Worker_dir)) + "-" + name,
		ready:    make(chan *Cgroup, CGROUP_RESERVE),
		recycled: make(chan *Cgroup, CGROUP_RESERVE),
		quit:     make(chan chan bool),
		nextID:   0,
	}

	// create cgroup
	path := pool.GroupPath()
	pool.printf("create %s", path)
	if err := syscall.Mkdir(path, 0700); err != nil {
		return nil, fmt.Errorf("Mkdir %s: %s", path, err)
	}

	// Make controllers available to child groups
	rpath := fmt.Sprintf("%s/cgroup.subtree_control", path)
	if err := ioutil.WriteFile(rpath, []byte("+pids +io +memory"), os.ModeAppend); err != nil {
		panic(fmt.Sprintf("Error writing to %s: %v", rpath, err))
	}

	go pool.cgTask()
	return pool, nil
}

// NewCgroup creates a new CGroup in the pool
func (pool *CgroupPool) NewCgroup() *Cgroup {
	pool.nextID++

	cg := &Cgroup{
		Name: fmt.Sprintf("cg-%d", pool.nextID),
		pool: pool,
	}

	path := cg.GroupPath()
	if err := syscall.Mkdir(path, 0700); err != nil {
		panic(fmt.Errorf("Mkdir %s: %s", path, err))
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
	path := cg.GroupPath()
	if err := syscall.Rmdir(path); err != nil {
		panic(fmt.Errorf("Rmdir %s: %s", path, err))
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
			// FIXME not possible in CG2?
			// cg.WriteInt("memory.failcnt", 0)
			cg.Unpause()
		default:
			t := common.T0("fresh-cgroup")
			cg = pool.NewCgroup()
			cg.WriteInt("pids.max", int64(common.Conf.Limits.Procs))
			cg.WriteInt("memory.swap.max", int64(common.Conf.Limits.Swappiness))
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

	// delete cgroup category
	gpath := pool.GroupPath()
	pool.printf("remove %s", gpath)
	if err := syscall.Rmdir(gpath); err != nil {
		panic(fmt.Errorf("Rmdir %s: %s", gpath, err))
	}
}

func (pool *CgroupPool) GetCg(memLimitMB int, moveMemCharge bool) *Cgroup {
	cg := <-pool.ready
	cg.setMemLimitMB(memLimitMB)
	/* FIXME not supported in CG2?
	   if moveMemCharge {
	       cg.WriteInt("memory.move_charge_at_immigrate", 1)
	   } else {
	       cg.WriteInt("memory.move_charge_at_immigrate", 0)
	   }*/
	return cg
}

// GroupPath returns the path to the Cgroup pool for OpenLambda
func (pool *CgroupPool) GroupPath() string {
	return fmt.Sprintf("/sys/fs/cgroup/%s", pool.Name)
}

// GroupPath returns the path to the Cgroup pool for OpenLambda
func (cg *Cgroup) GroupPath() string {
	return fmt.Sprintf("%s/%s", cg.pool.GroupPath(), cg.Name)
}

func (cg *Cgroup) MemoryEvents() map[string]int64 {
	result := map[string]int64{}
	path := cg.ResourcePath("memory.events")
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		entries := strings.Split(scanner.Text(), " ")
		key := entries[0]
		value, err := strconv.ParseInt(entries[1], 10, 64)
		if err != nil {
			panic(err)
		}
		result[key] = value
	}

	return result
}

// ResourcePath returns the path to a specific resource in this cgroup
func (cg *Cgroup) ResourcePath(resource string) string {
	return fmt.Sprintf("%s/%s/%s", cg.pool.GroupPath(), cg.Name, resource)
}

func (cg *Cgroup) TryWriteInt(resource string, val int64) error {
	return ioutil.WriteFile(cg.ResourcePath(resource), []byte(fmt.Sprintf("%d", val)), os.ModeAppend)
}

func (cg *Cgroup) TryWriteString(resource string, val string) error {
	return ioutil.WriteFile(cg.ResourcePath(resource), []byte(val), os.ModeAppend)
}

func (cg *Cgroup) WriteInt(resource string, val int64) {
	if err := cg.TryWriteInt(resource, val); err != nil {
		panic(fmt.Sprintf("Error writing %v to %s: %v", val, resource, err))
	}
}

func (cg *Cgroup) WriteString(resource string, val string) {
	if err := cg.TryWriteString(resource, val); err != nil {
		panic(fmt.Sprintf("Error writing %v to %s: %v", val, resource, err))
	}
}

func (cg *Cgroup) TryReadInt(resource string) (int64, error) {
	raw, err := ioutil.ReadFile(cg.ResourcePath(resource))
	if err != nil {
		return 0, err
	}
	val, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

func (cg *Cgroup) ReadInt(resource string) int64 {
	if val, err := cg.TryReadInt(resource); err != nil {
		panic(err)
	} else {
		return val
	}
}

func (cg *Cgroup) AddPid(pid string) error {
	err := ioutil.WriteFile(cg.ResourcePath("cgroup.procs"), []byte(pid), os.ModeAppend)
	if err != nil {
		return err
	}

	return nil
}

func (cg *Cgroup) setFreezeState(state int64) error {
	cg.WriteInt("cgroup.freeze", state)

	timeout := 5 * time.Second

	start := time.Now()
	for {
		freezerState, err := cg.TryReadInt("cgroup.freeze")
		if err != nil {
			return fmt.Errorf("failed to check self_freezing state :: %v", err)
		}

		if freezerState == state {
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
	usage := cg.ReadInt("memory.current")

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

	limitPath := cg.ResourcePath("memory.max")
	bytes := int64(mb) * 1024 * 1024
	cg.WriteInt("memory.max", bytes)

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
	return cg.setFreezeState(1)
}

func (cg *Cgroup) Unpause() error {
	return cg.setFreezeState(0)
}

func (cg *Cgroup) GetPIDs() ([]string, error) {
	procsPath := cg.ResourcePath("cgroup.procs")
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

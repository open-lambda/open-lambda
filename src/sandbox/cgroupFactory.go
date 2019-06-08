package sandbox

import (
	"fmt"
	"io/ioutil"
	"os"
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
			// TODO: make sure it is in a clean state (e.g., thawed)
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

func (cg *Cgroup) Init() {
	for _, resource := range cgroupList {
		if err := os.MkdirAll(cg.Path(resource, ""), 0700); err != nil {
			panic(err)
		}
	}
}

func (cg *Cgroup) Release() {
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

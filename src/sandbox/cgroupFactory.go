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

	"github.com/open-lambda/open-lambda/ol/config"
)

var cgroupList []string = []string{
	"blkio", "cpu", "devices", "freezer", "hugetlb",
	"memory", "perf_event", "systemd", "pids"}

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
		quit     chan chan bool
		nextId   int
	}

	func NewCgroupPool(name string) (*CgroupPool, error) {
		pool := &CgroupPool{
			Name:     path.Base(config.Conf.Worker_dir) + "-" + name,
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

    // get mem usage in bytes
	func (cg *Cgroup) getMemUsage() (int64, error) {
        usagePath := cg.Path("memory", "memory.usage_in_bytes")
        usageRaw, err := ioutil.ReadFile(usagePath)
        if err != nil {
            return -1, err
        }
        usage, err := strconv.ParseInt(strings.TrimSpace(string(usageRaw)), 10, 64)
        // if the mem usage cannot be read, return error
        if err != nil {
            return -1, err
        }
		// usageMB := int(usage / 1024 / 1024)
		return usage, nil
	}

    // set mem limit in bytes
	func (cg *Cgroup) getMemLimit() (int64, error) {
		limitPath := cg.Path("memory", "memory.limit_in_bytes")
		limitRaw, err := ioutil.ReadFile(limitPath)
		if err != nil {
			return -1, err
		}
		limit, err := strconv.ParseInt(strings.TrimSpace(string(limitRaw)), 10, 64)
		if err != nil {
			return -1, err
		}
		// limitMB := int(limit / 1024 / 1024)
		return limit, nil
	}

    // get mem limit in bytes
	func (cg *Cgroup) setMemLimit(newLimit int64) error {
		limitPath := cg.Path("memory", "memory.limit_in_bytes")
		// if mem limit cannot be set, return error
		if err := ioutil.WriteFile(limitPath, []byte(fmt.Sprintf("%d", newLimit)), os.ModeAppend); err != nil {
			return err
		}

		timeout := 5 * time.Second

		start := time.Now()
		for {
            memLimit, err := cg.getMemLimit()
			if err != nil {
				return fmt.Errorf("failed to get mem limit :: %v", err)
			}

			if memLimit == newLimit {
				return nil
			}

			if time.Since(start) > timeout {
				return fmt.Errorf("cgroup stuck on setting mem limit after %v", timeout)
			}

			time.Sleep(1 * time.Millisecond)
		}
    }

	// func (cg *Cgroup) changeMemLimit(pause bool) error {
	// 	limitPath := cg.Path("memory", "memory.limit_in_bytes")
	// 	oldLimitRaw, err := ioutil.ReadFile(limitPath)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	oldLimit, err := strconv.ParseInt(strings.TrimSpace(string(oldLimitRaw)), 10, 64)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	oldLimitMB := int(oldLimit / 1024 / 1024)

	// 	newLimitMB := 0
	// 	if pause {
	// 		usagePath := cg.Path("memory", "memory.usage_in_bytes")
	// 		usageRaw, err := ioutil.ReadFile(usagePath)
	// 		if err != nil {
	// 			return err
	// 		}
	// 		usage, err := strconv.ParseInt(strings.TrimSpace(string(usageRaw)), 10, 64)
	// 		// if the mem usage cannot be read, return error
	// 		if err != nil {
	// 			return err
	// 		}
	// 		// as memory limit will be rounded down to the nearest multiple of 4096 (page size)
	// 		newLimitMB = int((usage + 4096) / 1024 / 1024)
	// 	} else {
	// 		newLimitMB = config.Conf.Sock_cgroups.Max_mem_mb
	// 	}

	// 	// if mem limit cannot be set, return error
	// 	if err := ioutil.WriteFile(limitPath, []byte(fmt.Sprintf("%dM", newLimitMB)), os.ModeAppend); err != nil {
	// 		return err
	// 	}

	// 	// adjust available mem in mem pool, contradictory to the change in limit itself
	// 	cg.pool.memPool.adjustAvailableMB(oldLimitMB - newLimitMB) 

	// 	return nil
	// }

	func (cg *Cgroup) Pause() error {
        return cg.setFreezeState("FROZEN")
		// // if the cgroup cannot be frozen, return error
		// if err := cg.setFreezeState("FROZEN"); err != nil {
		// 	return err
		// }

		// // if the mem limit cannot be changed, return error
		// if err := cg.changeMemLimit(true); err != nil {
		// 	return err
		// }

		// cg.pool.printf("available mem after pause: %d", cg.pool.memPool.adjustAvailableMB(0))
		// cg.pool.printf("mem limit read from file after pause: %d", cg.readMemLimit())

		// return nil
	}


	func (cg *Cgroup) Unpause() error {
        return cg.setFreezeState("THAWED")
		// // if the mem limit cannot be changed, return error
		// if err := cg.changeMemLimit(false); err != nil {
		// 	return err
		// }
		// // if the cgroup cannot be thawed, return error
		// if err := cg.setFreezeState("THAWED"); err != nil {
		// 	return err
		// }

		// cg.pool.printf("available mem after unpause: %d", cg.pool.memPool.adjustAvailableMB(0))
		// cg.pool.printf("mem limit read from file after unpause: %d", cg.readMemLimit())

		// return nil
	}

	func (cg *Cgroup) GetPIDs() ([]string, error) {
		// we could use any cgroup resource type, as they should all
		// have the same procs
		procsPath := cg.Path("freezer", "cgroup.procs")
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

	// CG may be in any state before (Paused/Unpaused), will be empty and Unpaused after
	func (cg *Cgroup) KillAllProcs() error {
		if err := cg.Pause(); err != nil {
			return err
		}

		pids, err := cg.GetPIDs()
		if err != nil {
			return err
		}

		for _, pidStr := range pids {
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

		if err := cg.Unpause(); err != nil {
			return err
		}

		Loop:
		for i := 0; ; i++ {
			pids, err := cg.GetPIDs()
			if err != nil {
				return err
			} else if len(pids) == 0 {
				break Loop
			} else if i%1000 == 0 {
				cg.pool.printf("waiting for %d procs in %s to die", len(pids), cg.Name)
			}
			time.Sleep(1 * time.Millisecond)
		}

		return nil
	}

	func (cg *Cgroup) Init() {
		for _, resource := range cgroupList {
			path := cg.Path(resource, "")
			if err := syscall.Mkdir(path, 0700); err != nil {
				panic(fmt.Errorf("Mkdir %s: %s", path, err))
			}
		}

		// set limits based on config
		path := cg.Path("pids", "pids.max")
		err := ioutil.WriteFile(path, []byte(fmt.Sprintf("%d", config.Conf.Sock_cgroups.Max_procs)), os.ModeAppend)
		if err != nil {
			panic(fmt.Errorf("Error setting pids.max: %s", err))
		}

		path = cg.Path("memory", "memory.limit_in_bytes")
		err = ioutil.WriteFile(path, []byte(fmt.Sprintf("%dM", config.Conf.Sock_cgroups.Max_mem_mb)), os.ModeAppend)
		if err != nil {
			panic(fmt.Errorf("Error setting pids.max: %s", err))
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
		select {
		case cg.pool.recycled <- cg:
		default:
			cg.destroy()
		}
	}

	func (cg *Cgroup) destroy() {
		for _, resource := range cgroupList {
			path := cg.Path(resource, "")
			if err := syscall.Rmdir(path); err != nil {
				panic(fmt.Errorf("Rmdir %s: %s", path, err))
			}
		}
	}

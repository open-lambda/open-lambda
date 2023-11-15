package cgroups

import (
	"fmt"
	"io/ioutil"
	"log/slog"
	"os"
	"path"
	//"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/ol/common"
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

func NewCgroupPool(name string) (*CgroupPool, error) {
	pool := &CgroupPool{
		Name:     path.Base(path.Dir(common.Conf.Worker_dir)) + "-" + name,
		ready:    make(chan *CgroupImpl, CGROUP_RESERVE),
		recycled: make(chan *CgroupImpl, CGROUP_RESERVE),
		quit:     make(chan chan bool),
		nextID:   0,
	}

	// create cgroup
	groupPath := pool.GroupPath()
	pool.printf("INFO", "create %s", groupPath)
	if err := syscall.Mkdir(groupPath, 0700); err != nil {
		return nil, fmt.Errorf("Mkdir %s: %s", groupPath, err)
	}

	// Make controllers available to child groups
	rpath := fmt.Sprintf("%s/cgroup.subtree_control", groupPath)
	if err := ioutil.WriteFile(rpath, []byte("+pids +io +memory +cpu"), os.ModeAppend); err != nil {
		panic(fmt.Sprintf("Error writing to %s: %v", rpath, err))
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

	cg.printf("INFO", "created")
	return cg
}

// add ID to each log message so we know which logs correspond to
// which containers
func (pool *CgroupPool) printf(level string, format string, args ...any) {
	
	msg := fmt.Sprintf(format, args...)
	currLevel := common.Conf.Log.CgroupsLevel
	logger := slog.New(slog.Default().Handler())

	if (common.Conf.Log.LogFormat == "text") {
		logFilePath := path.Join(common.Conf.Log.Log_file_dir, "log.txt")
		f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			panic(fmt.Errorf("Cannot open log file at %s", logFilePath))
		}
		logger = slog.New(slog.NewTextHandler(f, nil))
		//defer f.Close()
	} else if (common.Conf.Log.LogFormat == "json") {
		logFilePath := path.Join(common.Conf.Log.Log_file_dir, "log.json")
		f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			panic(fmt.Errorf("Cannot open log file at %s", logFilePath))
		}
		logger = slog.New(slog.NewJSONHandler(f, nil))
		//defer f.Close()
	}
	
	if currLevel == "INFO" && level == currLevel {
		logger.Info(msg, "CGROUP POOL", pool.Name)

	} else if currLevel == "WARN" && level == currLevel {
		logger.Warn(msg, "CGROUP POOL", pool.Name)

	} else if currLevel == "ERROR" && level == currLevel {
		logger.Error(msg, "CGROUP POOL", pool.Name)
	}
	//log.Printf("%s [CGROUP POOL %s]", strings.TrimRight(msg, "\n"), pool.Name)
}

func (pool *CgroupPool) cgTask() {
	// we'll be sent this as part of the quit request
	var done chan bool

	// loop until we get the quit message
	pool.printf("INFO", "start creating/serving CGs")
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
			pool.printf("INFO", "received shutdown request")
			cg.Destroy()
			break Loop
		}
	}

	// empty queues, freeing all cgroups
	pool.printf("INFO", "empty queues and release CGs")
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

/// Destroy this entire cgroup pool
func (pool *CgroupPool) Destroy() {
	// signal cgTask, then wait for it to finish
	ch := make(chan bool)
	pool.quit <- ch
	<-ch

	// Destroy cgroup for this entire pool
	gpath := pool.GroupPath()
	pool.printf("INFO", "Destroying cgroup pool with path \"%s\"", gpath)
	for i := 100; i >= 0; i-- {
		if err := syscall.Rmdir(gpath); err != nil {
			if i == 0 {
				panic(fmt.Errorf("Rmdir %s: %s", gpath, err))
			} else {
				pool.printf("ERROR", "cgroup pool Rmdir failed, trying again in 5ms")
				time.Sleep(5 * time.Millisecond)
			}
		} else {
			break
		}
	}
}

func (pool *CgroupPool) GetCg(memLimitMB int, moveMemCharge bool, cpuPercent int) Cgroup {
	cg := <-pool.ready
	cg.SetMemLimitMB(memLimitMB)
	cg.SetCPUPercent(cpuPercent)

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

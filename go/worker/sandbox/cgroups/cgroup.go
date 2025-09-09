package cgroups

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/open-lambda/open-lambda/ol/common"
)

type CgroupImpl struct {
	name       string
	pool       *CgroupPool
	memLimitMB int
}

func (cg *CgroupImpl) printf(format string, args ...any) {
	if common.Conf.Trace.Cgroups {
		msg := fmt.Sprintf(format, args...)
		log.Printf("%s [CGROUP %s: %s]", strings.TrimRight(msg, "\n"), cg.pool.Name, cg.name)
	}
}

// Name returns the name of the cgroup.
func (cg *CgroupImpl) Name() string {
	return cg.name
}

// Release releases the cgroup back to the pool or destroys it if the pool is full.
func (cg *CgroupImpl) Release() {
	// if there's room in the recycled channel, add it there.
	// Otherwise, just delete it.
	if common.Conf.Features.Reuse_cgroups {
		for i := 100; i >= 0; i-- {
			pids, err := cg.GetPIDs()
			if err != nil {
				panic(err)
			} else if len(pids) > 0 {
				if i == 0 {
					panic(fmt.Errorf("Cannot release cgroup that contains processes: %v", pids))
				}

				cg.printf("cgroup Rmdir failed, trying again in 5ms")
				time.Sleep(5 * time.Millisecond)
			} else {
				break
			}
		}

		select {
		case cg.pool.recycled <- cg:
			cg.printf("release and recycle")
			return
		default:
		}
	}

	cg.printf("release and Destroy")
	cg.Destroy()
}

// Destroy destroys the cgroup.
func (cg *CgroupImpl) Destroy() {
	gpath := cg.GroupPath()
	cg.printf("Destroying cgroup with path \"%s\"", gpath)

	for i := 100; i >= 0; i-- {
		if err := syscall.Rmdir(gpath); err != nil {
			if i == 0 {
				panic(fmt.Errorf("Rmdir(2) %s: %s", gpath, err))
			}

			cg.printf("cgroup Rmdir failed, trying again in 5ms")
			time.Sleep(5 * time.Millisecond)
		} else {
			break
		}
	}
}

// GroupPath returns the path to the Cgroup pool for OpenLambda
func (cg *CgroupImpl) GroupPath() string {
	return fmt.Sprintf("%s/%s", cg.pool.GroupPath(), cg.name)
}

func (cg *CgroupImpl) MemoryEvents() map[string]int64 {
	result := map[string]int64{}
	groupPath := cg.ResourcePath("memory.events")
	f, err := os.Open(groupPath)
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
func (cg *CgroupImpl) ResourcePath(resource string) string {
	return fmt.Sprintf("%s/%s/%s", cg.pool.GroupPath(), cg.name, resource)
}

func (cg *CgroupImpl) TryWriteInt(resource string, val int64) error {
	return ioutil.WriteFile(cg.ResourcePath(resource), []byte(fmt.Sprintf("%d", val)), os.ModeAppend)
}

func (cg *CgroupImpl) TryWriteString(resource string, val string) error {
	return ioutil.WriteFile(cg.ResourcePath(resource), []byte(val), os.ModeAppend)
}

func (cg *CgroupImpl) WriteInt(resource string, val int64) {
	if err := cg.TryWriteInt(resource, val); err != nil {
		panic(fmt.Sprintf("Error writing %v to %s: %v", val, resource, err))
	}
}

func (cg *CgroupImpl) WriteString(resource string, val string) {
	if err := cg.TryWriteString(resource, val); err != nil {
		panic(fmt.Sprintf("Error writing %v to %s: %v", val, resource, err))
	}
}

func (cg *CgroupImpl) TryReadIntKV(resource string, key string) (int64, error) {
	raw, err := ioutil.ReadFile(cg.ResourcePath(resource))
	if err != nil {
		return 0, err
	}
	body := string(raw)
	lines := strings.Split(body, "\n")
	for i := 0; i <= len(lines); i++ {
		parts := strings.Split(lines[i], " ")
		if len(parts) == 2 && parts[0] == key {
			val, err := strconv.ParseInt(strings.TrimSpace(string(parts[1])), 10, 64)
			if err != nil {
				return 0, err
			}
			return val, nil
		}
	}
	return 0, fmt.Errorf("could not find key '%s' in file: %s", key, body)
}

func (cg *CgroupImpl) TryReadInt(resource string) (int64, error) {
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

func (cg *CgroupImpl) ReadInt(resource string) int64 {
	val, err := cg.TryReadInt(resource)

	if err != nil {
		panic(err)
	}

	return val
}

// AddPid adds a process ID to the cgroup.
func (cg *CgroupImpl) AddPid(pid string) error {
	err := ioutil.WriteFile(cg.ResourcePath("cgroup.procs"), []byte(pid), os.ModeAppend)
	if err != nil {
		return err
	}

	return nil
}

func (cg *CgroupImpl) setFreezeState(state int64) error {
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
			return fmt.Errorf("cgroup stuck on %v after %v (should be %v)", freezerState, timeout, state)
		}

		time.Sleep(1 * time.Millisecond)
	}
}

// get mem usage in MB
func (cg *CgroupImpl) GetMemUsageMB() int {
	usage := cg.ReadInt("memory.current")

	// round up to nearest MB
	mb := int64(1024 * 1024)
	return int((usage + mb - 1) / mb)
}

// get mem limit in MB
func (cg *CgroupImpl) GetMemLimitMB() int {
	return cg.memLimitMB
}

// set mem limit in MB
func (cg *CgroupImpl) SetMemLimitMB(mb int) {
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

// percent of a core
func (cg *CgroupImpl) SetCPUPercent(percent int) {
	period := 100000 // 100 ms
	quota := period * percent / 100
	cg.WriteString("cpu.max", fmt.Sprintf("%d %d", quota, period))
}

// Pause freezes processes in the cgroup.
func (cg *CgroupImpl) Pause() error {
	return cg.setFreezeState(1)
}

// Unpause unfreezes processes in the cgroup.
func (cg *CgroupImpl) Unpause() error {
	return cg.setFreezeState(0)
}

// GetPIDs returns the IDs of all processes running in this cgroup.
func (cg *CgroupImpl) GetPIDs() ([]string, error) {
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

// CgroupProcsPath returns the path to the cgroup.procs file.
func (cg *CgroupImpl) CgroupProcsPath() string {
	return cg.ResourcePath("cgroup.procs")
}

// KillAllProcs stops all processes inside the cgroup.
// Note, the CG most be paused beforehand
func (cg *CgroupImpl) KillAllProcs() {
	cg.WriteInt("cgroup.kill", 1)
}

// DebugString returns a string representation of the cgroup's state.
func (cg *CgroupImpl) DebugString() string {
	s := ""
	if pids, err := cg.GetPIDs(); err == nil {
		s += fmt.Sprintf("CGROUP PIDS: %s\n", strings.Join(pids, ", "))
	} else {
		s += fmt.Sprintf("CGROUP PIDS: unknown (%s)\n", err)
	}

	s += fmt.Sprintf("CGROUPS: %s\n", cg.ResourcePath("<RESOURCE>."))

	if state, err := ioutil.ReadFile(cg.ResourcePath("cgroup.freeze")); err == nil {
		s += fmt.Sprintf("FREEZE STATE: %s", state)
	} else {
		s += fmt.Sprintf("FREEZE STATE: unknown (%s)\n", err)
	}

	s += fmt.Sprintf("MEMORY USED: %d of %d MB\n",
		cg.GetMemUsageMB(), cg.GetMemLimitMB())

	if kills, err := cg.TryReadIntKV("memory.events", "oom_kill"); err == nil {
		s += fmt.Sprintf("OOM KILLS: %d\n", kills)
	} else {
		s += fmt.Sprintf("OOM KILLS: could not read because %d\n", err.Error())
	}
	return s
}

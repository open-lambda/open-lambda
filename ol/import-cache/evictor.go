package cache

/*
#include <sys/eventfd.h>
*/
import "C"

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
)

type Evictor struct {
	cm        *CacheManager
	limit     int
	eventfd   int
	usagePath string
}

//func NewEvictor(pkgfile, rootCID string, kb_limit int, full *bool) (*Evictor, error) {
func NewEvictor(cm *CacheManager, pkgfile, memCGroupPath string, kb_limit int) (*Evictor, error) {
	byte_limit := 1024 * kb_limit

	eventfd, err := C.eventfd(0, C.EFD_CLOEXEC)
	if err != nil {
		return nil, err
	}

	usagePath := filepath.Join(memCGroupPath, "memory.usage_in_bytes")
	usagefd, err := syscall.Open(usagePath, syscall.O_RDONLY, 0777)
	if err != nil {
		return nil, err
	}

	eventPath := filepath.Join(memCGroupPath, "cgroup.event_control")

	eventStr := fmt.Sprintf("'%d %d %d'", eventfd, usagefd, byte_limit)
	echo := exec.Command("echo", eventStr, ">", eventPath)
	if err = echo.Run(); err != nil {
		return nil, err
	}

	e := &Evictor{
		cm:        cm,
		limit:     byte_limit,
		eventfd:   int(eventfd),
		usagePath: usagePath,
		//full:      full,
	}

	return e, nil
}

func (e *Evictor) CheckUsage() {
	e.cm.mutex.Lock()
	defer e.cm.mutex.Unlock()

	usage := e.usage()
	if usage > e.limit {
		atomic.StoreInt32(e.cm.full, 1)
		e.evict()
	} else {
		atomic.StoreInt32(e.cm.full, 0)
	}
}

func (e *Evictor) usage() (usage int) {
	buf, err := ioutil.ReadFile(e.usagePath)
	if err != nil {
		return 0
	}

	str := strings.TrimSpace(string(buf[:]))
	usage, err = strconv.Atoi(str)
	if err != nil {
		panic(fmt.Sprintf("atoi failed: %v", err))
	}

	return usage
}

func (e *Evictor) evict() {
	servers := e.cm.servers
	idx := -1
	worst := float64(math.Inf(+1))

	for k := 1; k < len(servers); k++ {
		if servers[k].Children == 0 {
			if ratio := servers[k].Hits / servers[k].Size; ratio < worst {
				idx = k
				worst = ratio
			}
		}
	}

	if idx != -1 {
		// make sure no one else is using this one..
		victim := servers[idx]
		victim.Mutex.Lock()
		victim.Mutex.Unlock()

		e.cm.servers = append(servers[:idx], servers[idx+1:]...)
		go victim.Kill()
	} else {
		log.Printf("No victim found")
	}
}

package boss

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"sync"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

var tConf map[string]int
var tConf_lock sync.Mutex

func InitTestConf() error {
	tConf = make(map[string]int)
	for i := 1; i <= 32; i++ {
		key := "worker-" + strconv.Itoa(i)
		tConf[key] = 0
	}
	path := "worker_throughput.json"
	var content []byte

	content, err := json.MarshalIndent(tConf, "", "\t")
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(path, content, 0666); err != nil {
		return err
	}
	return nil
}

func WriteBack() error {
	path := "worker_throughput.json"
	var content []byte

	tConf_lock.Lock()
	content, err := json.MarshalIndent(tConf, "", "\t")
	if err != nil {
		return err
	}
	tConf_lock.Unlock()

	if err = ioutil.WriteFile(path, content, 0666); err != nil {
		return err
	}
	return nil
}

var boss_usage *os.File
var boss_log *log.Logger

func getStatus() {
	stat, err := linuxproc.ReadStat("/proc/stat")
	if err != nil {
		log.Fatal("stat read fail")
	}
	total := 0.0
	idle := 0.0
	total += float64(stat.CPUStatAll.User)
	total += float64(stat.CPUStatAll.Nice)
	total += float64(stat.CPUStatAll.System)
	total += float64(stat.CPUStatAll.Idle)
	total += float64(stat.CPUStatAll.IOWait)
	total += float64(stat.CPUStatAll.IRQ)
	total += float64(stat.CPUStatAll.SoftIRQ)
	total += float64(stat.CPUStatAll.Steal)
	total += float64(stat.CPUStatAll.Guest)
	total += float64(stat.CPUStatAll.GuestNice)

	idle += float64(stat.CPUStatAll.Idle)

	cpuPerc := ((total - idle) / total) * 100.0

	boss_log.Printf("cpu total: %f, cpu idle: %f, cpu usage percentage: %f%%", total, idle, cpuPerc)

	memInfo, err := linuxproc.ReadMemInfo("/proc/meminfo")
	if err != nil {
		log.Fatal("mem read fail")
	}
	memAva := float64(memInfo.MemAvailable)
	memTotal := float64(memInfo.MemTotal)
	memPerc := (memTotal - memAva) / float64(memTotal)
	memPerc = memPerc * 100.0
	boss_log.Printf("memory total: %f, memory available: %f, memory usage percentage: %f%%", memTotal, memAva, memPerc)
	boss_log.Println()
}

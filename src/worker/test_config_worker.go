package worker

import (
	"log"
	"os"

	linuxproc "github.com/c9s/goprocinfo/linux"
)

var worker_usage *os.File
var worker_log *log.Logger

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

	worker_log.Printf("cpu total: %f, cpu idle: %f, cpu usage percentage: %f%%", total, idle, cpuPerc)

	memInfo, err := linuxproc.ReadMemInfo("/proc/meminfo")
	if err != nil {
		log.Fatal("mem read fail")
	}
	memAva := float64(memInfo.MemFree)
	memTotal := float64(memInfo.MemTotal)
	memPerc := (memTotal - memAva) / float64(memTotal)
	memPerc = memPerc * 100.0
	worker_log.Printf("memory total: %f, memory available: %f, memory usage percentage: %f%%", memTotal, memAva, memPerc)
	worker_log.Println()
}

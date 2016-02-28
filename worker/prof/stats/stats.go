package main

import (
	"log"
	"os"
	"time"

	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

func main() {
	host, _ := os.LookupEnv("INFLUX_PORT_8086_TCP_ADDR")
	port, _ := os.LookupEnv("INFLUX_PORT_8086_TCP_PORT")
	log.Printf("connecting to influxdb at %s:%s\n", host, port)
	mgr := NewInfluxManager(host, port)
	for {
		recordLoad(mgr)
		recordMem(mgr)

		time.Sleep(50 * time.Millisecond)
	}
}

func recordLoad(mgr *InfluxManager) {
	l, _ := load.LoadAvg()

	fields := map[string]interface{}{
		"load1": l.Load1,
	}
	mgr.AddPointNow("load", fields)
}

func recordMem(mgr *InfluxManager) {
	v, _ := mem.VirtualMemory()

	fields := map[string]interface{}{
		"free":         v.Free,
		"used-percent": v.UsedPercent,
	}
	mgr.AddPointNow("memory", fields)
}

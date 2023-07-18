package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	default_outfile  = "resource_usage.csv"
	defualt_interval = 3 // in seconds
)

func main() {
	outfile := default_outfile
	interval := defualt_interval
	args := os.Args[1:]
	for i:=0; i<len(args) && len(args)%2==0; i++ {
		switch args[i] {
		case "-o":
			outfile = args[i+1]
			i++
		case "-t":
			var err error
			interval, err = strconv.Atoi(args[i+1])
			if err != nil {
				fmt.Println("Usage: go run resource_monitor.go -o [filename] -t [interval(sec)]")
			}
			i++	
		default:
			fmt.Println("Usage: go run resource_monitor.go -o [filename] -t [interval(sec)]")
			return
		}
	}

	csvFile, err := os.Create(outfile)
	if err != nil {
		fmt.Println("Error creating CSV file:", err)
		os.Exit(1)
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	err = csvWriter.Write([]string{"Timestamp", "CPU Usage (%)", "Memory Usage (MB)"})
	if err != nil {
		fmt.Println("Error writing CSV header:", err)
		os.Exit(1)
	}

	var prev_totalCPU, prev_idleCPU uint64
	for {
		timestamp := time.Now().Format("15:04:05")

		totalCPU, idleCPU, err := getCPU()
		if err != nil {
			fmt.Println("Error getting CPU usage:", err)
		}

		totalDelta := float64(totalCPU-prev_totalCPU)
		idleDelta := float64(idleCPU-prev_idleCPU)
		prev_totalCPU, prev_idleCPU = totalCPU, idleCPU
		cpuUsage := fmt.Sprintf("%.2f", 100*(totalDelta - idleDelta)/totalDelta)


		totalKb, availableKb, err := getMemory()
		if err != nil {
			fmt.Println("Error getting memory usage:", err)
		}

		memoryUsage := fmt.Sprintf("%.2f", float64(totalKb-availableKb)/1024)

		err = csvWriter.Write([]string{timestamp, cpuUsage, memoryUsage})
		if err != nil {
			fmt.Println("Error writing CSV data:", err)
		}

		csvWriter.Flush()
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func getCPU() (uint64, uint64, error) {
	contents, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(contents), "\n")
	fields := strings.Fields(lines[0])
	var totalCPU, idleCPU uint64
	for i := 1; i < len(fields); i++ {
		value, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		totalCPU += value

		if i == 4 {
			idleCPU = value
		}
	}
	return totalCPU, idleCPU, nil
}

func getMemory() (uint64, uint64, error) {
	contents, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}

	lines := strings.Split(string(contents), "\n")

	fields := strings.Fields(lines[0])
	totalKb, err := strconv.ParseUint(fields[1], 10, 64)
	
	fields = strings.Fields(lines[2])
	availableKb, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}

	return totalKb, availableKb, nil
}

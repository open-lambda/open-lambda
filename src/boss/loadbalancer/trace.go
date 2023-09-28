package loadbalancer

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
)

var Traces *TraceList

type Trace struct {
	Deps []string `json:"deps"`
	Name string   `json:"name"`
	Top  []string `json:"top"`
	Type string   `json:"type"`
}

type TraceList struct {
	Data []Trace
}

func LoadTrace() *TraceList {
	filePath := "/home/azureuser/open-lambda/dep-trace.json"
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %s", err)
	}
	defer file.Close()

	var data []Trace
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var record Trace
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			log.Fatalf("Error parsing line as JSON: %s", err)
		}
		data = append(data, record)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading file: %s", err)
	}

	res := &TraceList{
		Data: data,
	}
	return res
}

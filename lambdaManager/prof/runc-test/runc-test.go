package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
)

func main() {
	var numRequests int

	if len(os.Args) < 2 {
		fmt.Println("Usage: ./runc-test <num_requests>")
		return
	}

	num, err := strconv.Atoi(os.Args[1])
	if err == nil {
		numRequests = num
	} else {
		log.Printf("bad num_requests: %v", err)
	}

	timesToStart := make([]time.Duration, numRequests)
	timesRoundTrip := make([]time.Duration, numRequests)

	for i := 0; i < numRequests; i++ {
		duration := time.Second * 1
		time.Sleep(duration)

		cmd := exec.Command("runc", "start")
		startTime := time.Now()
		out, err := cmd.Output()
		endTime := time.Now()

		if err != nil {
			log.Fatal("cmd failed:", err)
		}

		type Timestamp struct {
			Time string `json:"startup"`
		}

		// parse body
		var startup Timestamp
		err = json.Unmarshal((out), &startup)
		if err != nil {
			log.Printf("timestamp unmarshal failed on request %d", i)
			continue
		}

		form := "2006-01-02 15:04:05.999999999 +0000 UTC"
		containerStartTime, err := time.Parse(form, startup.Time)
		if err != nil {
			log.Printf("failed to parse timestamp %s for request %d\n", startup.Time, i)
			continue
		}

		// create 'timeToStart' value.
		// This is the time from req, to lambda running
		timeToStart := containerStartTime.Sub(startTime)
		timesToStart = append(timesToStart, timeToStart)
		log.Printf("timeToStart: %v\n", timeToStart)

		// create 'roundTripTime'
		// This is the time from req, to resp
		roundTripTime := endTime.Sub(startTime)
		timesRoundTrip = append(timesRoundTrip, roundTripTime)
		log.Printf("totalRuntime: %v\n", roundTripTime)
	}

	// average all the times
	var avgTimeToStart float64
	avgTimeToStart = 0
	for _, dur := range timesToStart {
		avgTimeToStart += dur.Seconds()
	}
	avgTimeToStart /= float64(numRequests)

	var avgRoundTrip float64
	avgRoundTrip = 0
	for _, dur := range timesRoundTrip {
		avgRoundTrip += dur.Seconds()
	}
	avgRoundTrip /= float64(numRequests)

	// log.Printf("avg time to container start (ms): %d\n", avgTimeToStart/(1000*1000))
	// log.Printf("avg round trip time (ms): %d\n", avgRoundTrip/(1000*1000))
	log.Printf("avg time to container start (s): %g\n", avgTimeToStart)
	log.Printf("avg total time (s): %g\n", avgRoundTrip)
}

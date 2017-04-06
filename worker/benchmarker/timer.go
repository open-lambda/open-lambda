package benchmarker

import (
	"fmt"
	"os"
	"time"
)

type Timer struct {
	logfile    *os.File
	start_time int64
	name       string
	unit       string
}

func (t *Timer) Start() {
	t.start_time = time.Now().UnixNano()
}

func (t *Timer) End() {
	end_time := time.Now().UnixNano()
	diff := end_time - t.start_time
	if t.unit == "us" {
		diff /= 1000
	} else if t.unit == "ms" {
		diff /= 1000000
	}
	fmt.Fprintf(t.logfile, "%s: %d %s\n", t.name, diff, t.unit)
}

func (t *Timer) Error(message string) {
	end_time := time.Now().UnixNano()
	diff := end_time - t.start_time
	if t.unit == "us" {
		diff /= 1000
	} else if t.unit == "ms" {
		diff /= 1000000
	}
	fmt.Fprintf(t.logfile, "ERROR:%s %d %s\n", message, diff, t.unit)
}

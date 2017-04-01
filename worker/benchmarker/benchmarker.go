package benchmarker

import (
	"log"
	"os"
)

type Benchmarker struct {
	logfile *os.File
}

var b *Benchmarker = nil

func CreateBenchmarkerSingleton(log_file string) {
	f, err := os.OpenFile(log_file, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening benchmark log file: %v", err)
	}
	b = &Benchmarker{
		logfile: f,
	}
}

func GetBenchmarker() *Benchmarker {
	return b
}

func (b *Benchmarker) CreateTimer(name string, unit string) *Timer {
	return &Timer{
		logfile: b.logfile,
		name:    name,
		unit:    unit,
	}
}

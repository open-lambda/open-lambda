package common

import (
	"bytes"
	"container/list"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"
	"log"
)

type RollingAvg struct {
	size int
	nums *list.List
	sum  int
	Avg  int
}

func NewRollingAvg(size int) *RollingAvg {
	return &RollingAvg{
		size: size,
		nums: list.New(),
		sum:  0,
		Avg:  0,
	}
}

func (r *RollingAvg) Add(num int) {
	r.sum += num
	r.nums.PushFront(num)
	if r.nums.Len() > r.size {
		r.sum -= r.nums.Back().Value.(int)
		r.nums.Remove(r.nums.Back())
	}
	r.Avg = r.sum / r.nums.Len()
}

// process-global stats server

type msLatencyMsg struct {
	name string
	x    int64
}

type snapshotMsg struct {
	stats map[string]int64
	done  chan bool
}

var initOnce sync.Once
var statsChan chan any = make(chan any, 256)

func initTaskOnce() {
	initOnce.Do(func() {
		go statsTask()
	})
}

func statsTask() {
	msCounts := make(map[string]int64)
	msSums := make(map[string]int64)

	for raw := range statsChan {
		switch msg := raw.(type) {
		case *msLatencyMsg:
			msCounts[msg.name] += 1
			msSums[msg.name] += msg.x
		case *snapshotMsg:
			for k, cnt := range msCounts {
				msg.stats[k+".cnt"] = cnt
				msg.stats[k+".ms-avg"] = msSums[k] / cnt
			}
			msg.done <- true
		default:
			panic(fmt.Sprintf("unkown type: %T", msg))
		}
	}
}

func record(name string, x int64) {
	initTaskOnce()
	statsChan <- &msLatencyMsg{name, x}
}

func SnapshotStats() map[string]int64 {
	initTaskOnce()
	stats := make(map[string]int64)
	done := make(chan bool)
	statsChan <- &snapshotMsg{stats, done}
	<-done
	return stats
}

type Latency struct {
	name         string
	t0           time.Time
	Milliseconds int64
}

// record start time
func T0(name string) *Latency {
	return &Latency{
		name: name,
		t0:   time.Now(),
	}
}

// measure latency to end time, and record it
func (l *Latency) T1() {
	l.Milliseconds = int64(time.Now().Sub(l.t0)) / 1000000
	if l.Milliseconds < 0 {
		panic("negative latency")
	}
	record(l.name, l.Milliseconds)

	// make sure we didn't double record
	var zero time.Time
	if l.t0 == zero {
		panic("double counted stat for " + l.name)
	}
	l.t0 = zero

	if Conf.Trace.Latency {
		log.Printf("%s=%d ms", l.name, l.Milliseconds)
	}
}

// start measuring a sub latency
func (l *Latency) T0(name string) *Latency {
	return T0(l.name + "/" + name)
}

// https://blog.sgmansfield.com/2015/12/goroutine-ids/
//
// this is for debugging only (e.g., if we want to correlate a trace
// with a core dump
func GetGoroutineID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

func Max(x int, y int) int {
	if x > y {
		return x
	}

	return y
}

func Min(x int, y int) int {
	if x < y {
		return x
	}

	return y
}

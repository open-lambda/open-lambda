package stats

import (
	"container/list"
	"fmt"
	"sync"
	"time"
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

type recordMsg struct {
	name string
	x    int64
}

type snapshotMsg struct {
	stats map[string]int64
	done  chan bool
}

var initOnce sync.Once
var statsChan chan interface{} = make(chan interface{}, 256)

func initTaskOnce() {
	initOnce.Do(func() {
		go statsTask()
	})
}

func statsTask() {
	counts := make(map[string]int64)
	sums := make(map[string]int64)

	for raw := range statsChan {
		switch msg := raw.(type) {
		case *recordMsg:
			counts[msg.name] += 1
			sums[msg.name] += msg.x
		case *snapshotMsg:
			for k, cnt := range counts {
				msg.stats[k+".cnt"] = cnt
				msg.stats[k+".avg"] = sums[k] / cnt
			}
			msg.done <- true
		default:
			panic(fmt.Sprintf("unkown type: %T", msg))
		}
	}
}

func Record(name string, x int64) {
	initTaskOnce()
	statsChan <- &recordMsg{name, x}
}

func Snapshot() map[string]int64 {
	initTaskOnce()
	stats := make(map[string]int64)
	done := make(chan bool)
	statsChan <- &snapshotMsg{stats, done}
	<-done
	return stats
}

type Latency struct {
	name string
	t0   time.Time
}

// record start time
func T0(name string) Latency {
	return Latency{
		name: name,
		t0:   time.Now(),
	}
}

// measure latency to end time, and record it
func (l Latency) T1() {
	ms := int64(time.Now().Sub(l.t0)) / 1000000
	if ms < 0 {
		panic("negative latency")
	}
	Record(l.name+":ms", ms)

	// make sure we didn't double record
	var zero time.Time
	if l.t0 == zero {
		panic("double counted stat for " + l.name)
	}
	l.t0 = zero
}

// start measuring a sub latency
func (l Latency) T0(name string) Latency {
	return T0(l.name + "/" + name)
}

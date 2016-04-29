package turnip

import "time"

const (
	defaultName = "deafult"
)

type Turnip struct {
	start time.Time

	times []time.Duration

	avg   float64
	count float64
}

// Start a timer
func (t *Turnip) Start() {
	t.start = time.Now()
}

// Stop a timer, and update stats
func (t *Turnip) Stop() {
	dur := time.Since(t.start)

	t.times = append(t.times, dur)
	t.count++

	// update average a1 = a0 + (t1 - a1)/n
	t.avg = t.avg + ((float64(dur.Nanoseconds()) - t.avg) / t.count)
}

// Get the average timer duration in Milliseconds
func (t *Turnip) AverageMs() float64 {
	ms := t.avg / (float64(time.Millisecond))

	return ms
}

// Create a new timer
func NewTurnip() *Turnip {
	t := &Turnip{
		times: make([]time.Duration, 0),
		count: 0,
	}

	return t
}

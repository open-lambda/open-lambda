package stats

import (
	"container/list"
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

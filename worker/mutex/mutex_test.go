package mutex

import (
	"testing"
	"time"
)

const (
	testThreads = 10
	sleepTime   = 10
)

func TestRlock(t *testing.T) {
	mtx := NewMutex()

	for i := 0; i < testThreads; i++ {
		go func() {
			for itr := 0; itr < 1000; itr++ {
				mtx.RLock()
				time.Sleep(sleepTime * time.Millisecond)
				mtx.RUnlock()
			}
		}()
	}
}

// Ensure write lock locks out readers and writers
func TestWLockLockout(t *testing.T) {
	smallTime := 10 * time.Millisecond

	mtx := NewMutex()

	mtx.Lock()

	// try to take a read lock
	gotLock := make(chan struct{})
	timer := time.NewTimer(smallTime)
	go func() {
		mtx.RLock()
		gotLock <- struct{}{}
	}()
	select {
	case <-gotLock:
		timer.Stop()
		t.Errorf("Got read lock while write held!\n")

	case <-timer.C:
	}

	// try to take a write lock
	gotLock = make(chan struct{})
	timer = time.NewTimer(smallTime)
	go func() {
		mtx.Lock()
		gotLock <- struct{}{}
	}()
	select {
	case <-gotLock:
		timer.Stop()
		t.Errorf("Got write lock while write held!\n")

	case <-timer.C:
	}

	// See if read lock continues, after write unlocked
	gotLock = make(chan struct{})
	timer = time.NewTimer(smallTime)
	go func() {
		mtx.RLock()
		gotLock <- struct{}{}
	}()

	time.Sleep(100 * time.Millisecond)
	mtx.Unlock()

	select {
	case <-gotLock:
		timer.Stop()
	case <-time.After(smallTime):
		t.Errorf("Failed to hand off read lock!\n")
	}

}

// Allow lots of readers
func TestMaxRlock(t *testing.T) {
	timeout := 10 * time.Second
	smallTime := 10 * time.Millisecond

	mtx := NewMutex()

	done := make(chan int)

	// take RLocks until failure
	go func() {
		count := 0
		gotLock := make(chan struct{})
		for {
			timer := time.NewTimer(smallTime)
			go func() {
				mtx.RLock()
				gotLock <- struct{}{}
			}()
			select {
			case <-gotLock:
				timer.Stop()
				count++

			case <-timer.C:
				done <- count
			}
			if count == 10000 {
				done <- count
				return
			}
		}
	}()

	var count int
	select {
	case count = <-done:
		if count != 10000 {
			t.Errorf("failed after %d rlocks. Expected 10000\n", count)
		}
		// pass
		return
	case <-time.After(timeout):
		t.Errorf("failed to finish after %ds\n", timeout)
	}
	close(done)
}

func TestReadUnlockFuncNoRun(t *testing.T) {
	smallTime := 10 * time.Millisecond
	mtx := NewMutex()

	// create multiple readers
	mtx.RLock()
	mtx.RLock()

	timer := time.NewTimer(smallTime)
	wasRun := make(chan struct{}, 1)
	ret := mtx.RUnlockFunc(func() {
		wasRun <- struct{}{}
		return
	})

	select {
	case <-wasRun:
		timer.Stop()
		t.Fatalf("user func run with reader still reading!\n")
	case <-timer.C:
	}

	if ret {
		t.Fatalf("RUnlockFunc returned true with readers stil reading!\n")
	}
}

func TestReadUnlockFuncRun(t *testing.T) {
	smallTime := 10 * time.Millisecond
	mtx := NewMutex()

	// create single reader
	mtx.RLock()

	timer := time.NewTimer(smallTime)
	wasRun := make(chan struct{}, 1)
	ret := mtx.RUnlockFunc(func() {
		wasRun <- struct{}{}
		return
	})

	select {
	case <-wasRun:
		timer.Stop()
	case <-timer.C:
		t.Fatalf("user func never run with no other readers!\n")
	}

	if !ret {
		t.Fatalf("RUnlockFunc returned false with no other readers!\n")
	}
}

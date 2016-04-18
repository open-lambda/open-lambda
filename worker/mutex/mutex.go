package mutex

import "sync"

const (
	maxReaders = 10000
)

// Adapted from http://play.golang.org/p/YXAHy5kBfD
type Mutex struct {
	l *sync.RWMutex

	countLock  *sync.Mutex
	numReaders int
}

// lock writer
// block if readers or writers
func (m *Mutex) Lock() {
	m.l.Lock()
}

// unlock writer
func (m *Mutex) Unlock() {
	m.l.Unlock()
}

// lock reader
func (m *Mutex) RLock() {
	m.l.RLock()

	m.countLock.Lock()
	m.numReaders++
	m.countLock.Unlock()
}

// unlock reader
func (m *Mutex) RUnlock() {
	m.countLock.Lock()
	m.numReaders--
	m.countLock.Unlock()

	m.l.RUnlock()
}

// unlock reader
// Other readers cannot take lock until user func returns
// Returns true if userFunc run, otherwise false
func (m *Mutex) RUnlockFunc(userFunc func()) bool {
	ret := false

	m.countLock.Lock()
	m.numReaders--
	if m.numReaders == 0 {
		userFunc()
		ret = true
	}
	m.countLock.Unlock()

	m.l.RUnlock()

	return ret
}

func NewMutex() *Mutex {
	return &Mutex{
		l:         &sync.RWMutex{},
		countLock: &sync.Mutex{},
	}
}

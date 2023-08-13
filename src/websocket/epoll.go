// This file includes code derived from the mailru/easygo project,
// which was released under the MIT License by s.kamardin.
// Copyright (c) 2023 Sergey Kamardin
//
// Original License:
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package websocket

import (
	"fmt"
	"log"
	"net"
	"reflect"
	"sync"
	"syscall"

	"golang.org/x/sys/unix"
)

// EpollEvent represents epoll events configuration bit mask.
type EpollEvent uint32

var (
	// ErrClosed is returned by Poller methods to indicate that instance is
	// closed and operation could not be processed.
	ErrClosed = fmt.Errorf("poller instance is closed")

	ErrRegistered = fmt.Errorf("file descriptor is already registered in poller instance")

	ErrNotRegistered = fmt.Errorf("file descriptor was not registered before in poller instance")
)

// EpollEvents that are mapped to epoll_event.events possible values.
const (
	EPOLLIN      = unix.EPOLLIN
	EPOLLOUT     = unix.EPOLLOUT
	EPOLLRDHUP   = unix.EPOLLRDHUP
	EPOLLPRI     = unix.EPOLLPRI
	EPOLLERR     = unix.EPOLLERR
	EPOLLHUP     = unix.EPOLLHUP
	EPOLLET      = unix.EPOLLET
	EPOLLONESHOT = unix.EPOLLONESHOT

	// _EPOLLCLOSED is a special EpollEvent value the receipt of which means
	// that the epoll instance is closed.
	_EPOLLCLOSED = 0x20
)

// String returns a string representation of EpollEvent.
func (evt EpollEvent) String() (str string) {
	name := func(event EpollEvent, name string) {
		if evt&event == 0 {
			return
		}
		if str != "" {
			str += "|"
		}
		str += name
	}

	name(EPOLLIN, "EPOLLIN")
	name(EPOLLOUT, "EPOLLOUT")
	name(EPOLLRDHUP, "EPOLLRDHUP")
	name(EPOLLPRI, "EPOLLPRI")
	name(EPOLLERR, "EPOLLERR")
	name(EPOLLHUP, "EPOLLHUP")
	name(EPOLLET, "EPOLLET")
	name(EPOLLONESHOT, "EPOLLONESHOT")
	name(_EPOLLCLOSED, "_EPOLLCLOSED")

	return
}

// Epoll represents single epoll instance.
type Epoll struct {
	mu sync.RWMutex

	fd       int
	eventFd  int
	closed   bool
	waitDone chan struct{}

	callbacks map[int]func(EpollEvent)
}

// EpollConfig contains options for Epoll instance configuration.
type EpollConfig struct {
	// OnWaitError will be called from goroutine, waiting for events.
	OnWaitError func(error)
}

func (c *EpollConfig) withDefaults() (config EpollConfig) {
	if c != nil {
		config = *c
	}
	if config.OnWaitError == nil {
		config.OnWaitError = defaultOnWaitError
	}
	return config
}

// CreateEpoll creates new epoll instance.
// It starts the wait loop in separate goroutine.
func CreateEpoll(c *EpollConfig) (*Epoll, error) {
	config := c.withDefaults()

	fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}

	r0, _, errno := unix.Syscall(unix.SYS_EVENTFD2, 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	eventFd := int(r0)

	// Set finalizer for write end of socket pair to avoid data races when
	// closing Epoll instance and EBADF errors on writing ctl bytes from callers.
	err = unix.EpollCtl(fd, unix.EPOLL_CTL_ADD, eventFd, &unix.EpollEvent{
		Events: unix.EPOLLIN,
		Fd:     int32(eventFd),
	})
	if err != nil {
		unix.Close(fd)
		unix.Close(eventFd)
		return nil, err
	}

	ep := &Epoll{
		fd:        fd,
		eventFd:   eventFd,
		callbacks: make(map[int]func(EpollEvent)),
		waitDone:  make(chan struct{}),
	}

	// Run wait loop.
	go ep.wait(config.OnWaitError)

	return ep, nil
}

// closeBytes used for writing to eventfd.
var closeBytes = []byte{1, 0, 0, 0, 0, 0, 0, 0}

// Close stops wait loop and closes all underlying resources.
func (ep *Epoll) Close() (err error) {
	ep.mu.Lock()
	{
		if ep.closed {
			ep.mu.Unlock()
			return ErrClosed
		}
		ep.closed = true

		if _, err = unix.Write(ep.eventFd, closeBytes); err != nil {
			ep.mu.Unlock()
			return
		}
	}
	ep.mu.Unlock()

	<-ep.waitDone

	if err = unix.Close(ep.eventFd); err != nil {
		return
	}

	ep.mu.Lock()
	// Set callbacks to nil preventing long mu.Lock() hold.
	// This could increase the speed of retreiving ErrClosed in other calls to
	// current epoll instance.
	// Setting callbacks to nil is safe here because no one should read after
	// closed flag is true.
	callbacks := ep.callbacks
	ep.callbacks = nil
	ep.mu.Unlock()

	for _, cb := range callbacks {
		if cb != nil {
			cb(_EPOLLCLOSED)
		}
	}

	return
}

// Add adds fd to epoll set with given events.
// Callback will be called on each received event from epoll.
// Note that _EPOLLCLOSED is triggered for every cb when epoll closed.
func (ep *Epoll) Add(client *Client, events EpollEvent, cb func(EpollEvent)) (err error) {
	conn := client.conn
	// Extract file descriptor associated with the connection
	// then store it in the client for later use.
	fd := getFD(conn)
	client.fd = fd

	ev := &unix.EpollEvent{
		Events: uint32(events),
		Fd:     int32(fd),
	}

	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.closed {
		return ErrClosed
	}
	if _, has := ep.callbacks[fd]; has {
		return ErrRegistered
	}
	ep.callbacks[fd] = cb

	return unix.EpollCtl(ep.fd, unix.EPOLL_CTL_ADD, fd, ev)
}

// Remove removes a client from the epoll set.
func (ep *Epoll) Remove(client *Client) error {
	fd := client.fd
	return ep.remove(fd)
}

// Del removes fd from epoll set.
func (ep *Epoll) remove(fd int) (err error) {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.closed {
		return ErrClosed
	}
	if _, ok := ep.callbacks[fd]; !ok {
		return ErrNotRegistered
	}

	delete(ep.callbacks, fd)

	return unix.EpollCtl(ep.fd, unix.EPOLL_CTL_DEL, fd, nil)
}

// Mod sets to listen events on fd.
func (ep *Epoll) Mod(fd int, events EpollEvent) (err error) {
	ev := &unix.EpollEvent{
		Events: uint32(events),
		Fd:     int32(fd),
	}

	ep.mu.RLock()
	defer ep.mu.RUnlock()

	if ep.closed {
		return ErrClosed
	}
	if _, ok := ep.callbacks[fd]; !ok {
		return ErrNotRegistered
	}

	return unix.EpollCtl(ep.fd, unix.EPOLL_CTL_MOD, fd, ev)
}

const (
	maxWaitEventsBegin = 1024
	maxWaitEventsStop  = 32768
)

func (ep *Epoll) wait(onError func(error)) {
	defer func() {
		if err := unix.Close(ep.fd); err != nil {
			onError(err)
		}
		close(ep.waitDone)
	}()

	events := make([]unix.EpollEvent, maxWaitEventsBegin)
	callbacks := make([]func(EpollEvent), 0, maxWaitEventsBegin)

	for {
		n, err := unix.EpollWait(ep.fd, events, -1)
		if err != nil {
			if temporaryErr(err) {
				continue
			}
			onError(err)
			return
		}

		callbacks = callbacks[:n]

		ep.mu.RLock()
		for i := 0; i < n; i++ {
			fd := int(events[i].Fd)
			if fd == ep.eventFd { // signal to close epoll
				ep.mu.RUnlock()
				return
			}
			callbacks[i] = ep.callbacks[fd]
		}
		ep.mu.RUnlock()

		for i := 0; i < n; i++ {
			if cb := callbacks[i]; cb != nil {
				cb(EpollEvent(events[i].Events))
				callbacks[i] = nil
			}
		}

		if n == len(events) && n*2 <= maxWaitEventsStop {
			events = make([]unix.EpollEvent, n*2)
			callbacks = make([]func(EpollEvent), 0, n*2)
		}
	}
}

func getFD(conn net.Conn) int {
	tcpConn := reflect.Indirect(reflect.ValueOf(conn)).FieldByName("conn")
	fdVal := tcpConn.FieldByName("fd")
	pfdVal := reflect.Indirect(fdVal).FieldByName("pfd")

	return int(pfdVal.FieldByName("Sysfd").Int())
}

func defaultOnWaitError(err error) {
	log.Printf("epoll: wait loop error: %s", err)
}

func temporaryErr(err error) bool {
	errno, ok := err.(syscall.Errno)
	if !ok {
		return false
	}
	return errno.Temporary()
}

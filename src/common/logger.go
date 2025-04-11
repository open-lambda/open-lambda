package common

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path"
	"sync"
	"sync/atomic"

	"github.com/urfave/cli/v2"
)

var defaultThreadSafeWriter atomic.Pointer[ThreadSafeWriter]
var DefaultSystemLogger *slog.Logger

func init() {
	defaultThreadSafeWriter.Store(newDefaultThreadSafeWriter(os.Stderr))
	log.SetOutput(defaultThreadSafeWriter.Load())
}

// Wrapper for a writer to introduce a shared lock at the intermediate level
type ThreadSafeWriter struct {
	w  io.Writer
	mu *sync.Mutex
}

// Write method so it can be considered a writer by log slog
func (ts *ThreadSafeWriter) Write(buf []byte) (int, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.w.Write(buf)
}

// Clones the current writer so that it shares a mutex but a new writer
func (ts *ThreadSafeWriter) clone() *ThreadSafeWriter {
	return &ThreadSafeWriter{
		w:  ts.w,
		mu: ts.mu,
	}
}

func (ts *ThreadSafeWriter) withWriter(w io.Writer) *ThreadSafeWriter {
	ts2 := ts.clone()
	ts2.mu.Lock()
	defer ts2.mu.Unlock()
	ts2.w = w
	return ts2
}

func newDefaultThreadSafeWriter(w io.Writer) *ThreadSafeWriter {
	return &ThreadSafeWriter{
		w:  w,
		mu: &sync.Mutex{},
	}
}

func NewThreadSafeWriter(w io.Writer) *ThreadSafeWriter {
	ts2 := defaultThreadSafeWriter.Load().withWriter(w)
	return ts2
}

// Taken from PR #203
func ParseLevelString(conf string) (*slog.LevelVar, error) {
	level := new(slog.LevelVar)
	if conf == "INFO" {
		level.Set(slog.LevelInfo)
	} else if conf == "WARN" {
		level.Set(slog.LevelWarn)
	} else if conf == "ERROR" {
		level.Set(slog.LevelError)
	} else if conf == "" {
		// logger is disabled
		level.Set(slog.LevelError + 1)
	} else {
		return level, fmt.Errorf("Unknown log level: %s", conf)
	}
	return level, nil
}

// Load Cgroups logger (Also taken from PR #203)
func LoadCgroupLogger(ilevel string) (*slog.Logger, error) {
	level, err := ParseLevelString(ilevel)
	if err != nil {
		return slog.Default(), err
	}
	if Conf.Trace.Enable_JSON == true {
		olpath, err := GetOlPath(&cli.Context{})
		if err != nil {
			return slog.Default(), err
		}
		logFilePath := path.Join(olpath, "worker.json")
		f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			panic(fmt.Errorf("Cannot open log file at %s%", logFilePath))
		}
		logger := slog.New(slog.NewJSONHandler(NewThreadSafeWriter(f), &slog.HandlerOptions{Level: level}))
		return logger, nil
	} else {
		logger := slog.New(slog.NewTextHandler(NewThreadSafeWriter(os.Stdout), &slog.HandlerOptions{Level: level}))
		return logger, nil
	}
}

package common

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path"
	"sync"
	"runtime"

	"github.com/urfave/cli/v2"
)

type OLHandler struct {
	level   slog.Leveler
	goas 	[]groupOrAttrs
	mu 		*sync.Mutex
	out 	io.Writer
}

func NewOLHandler(level slog.Leveler, out io.Writer) *OLHandler {
	if level == nil {
		// default to INFO (might not actually need this)
		level = slog.LevelInfo
	}
	return &OLHandler{level, nil, &sync.Mutex{}, out}
}

// Enabled implements Handler.Enabled by reporting whether
// level is at least as large as h's level.
func (h *OLHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

// Handle implements Handler.Handle.
func (h *OLHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)
	if !r.Time.IsZero() {
		buf = h.appendAttr(buf, slog.Time(slog.TimeKey, r.Time), 0)
	}
	buf = h.appendAttr(buf, slog.Any(slog.LevelKey, r.Level), 0)
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		buf = h.appendAttr(buf, slog.String(slog.SourceKey, fmt.Sprintf("%s:%d", f.File, f.Line)), 0)
	}
	buf = h.appendAttr(buf, slog.String(slog.MessageKey, r.Message), 0)
	indentLevel := 0
	// Handle state from WithGroup and WithAttrs.
	goas := h.goas
	if r.NumAttrs() == 0 {
		// If the record has no Attrs, remove groups at the end of the list; they are empty.
		for len(goas) > 0 && goas[len(goas)-1].group != "" {
			goas = goas[:len(goas)-1]
		}
	}
	for _, goa := range goas {
		if goa.group != "" {
			buf = fmt.Appendf(buf, "%*s%s: ", indentLevel*4, "", goa.group)
			indentLevel++
		} else {
			for _, a := range goa.attrs {
				buf = h.appendAttr(buf, a, indentLevel)
			}
		}
	}
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a, indentLevel)
		return true
	})
	buf = append(buf, "\n"...)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

func (h *OLHandler) appendAttr(buf []byte, a slog.Attr, indentLevel int) []byte {
	// Resolve the Attr's value before doing anything else.
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return buf
	}
	// Indent 4 spaces per level.
	buf = fmt.Appendf(buf, "%*s", indentLevel*4, "")
	switch a.Value.Kind() {
	case slog.KindString:
		// Quote string values, to make them easy to parse.
		buf = fmt.Appendf(buf, "%s: %q ", a.Key, a.Value.String())
	case slog.KindTime:
		// Currently using a time format that matches the old log's format. Can change to any preference.
		buf = fmt.Appendf(buf, "%s ", a.Value.Time().Format("2000/01/01 00:00:00.999999"))
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return buf
		}
		// If the key is non-empty, write it out and indent the rest of the attrs.
		// Otherwise, inline the attrs.
		if a.Key != "" {
			buf = fmt.Appendf(buf, "%s: ", a.Key)
			indentLevel++
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga, indentLevel)
		}
	default:
		buf = fmt.Appendf(buf, "%s: %s ", a.Key, a.Value)
	}
	return buf
}

// groupOrAttrs holds either a group name or a list of slog.Attrs.
type groupOrAttrs struct {
	group string      // group name if non-empty
	attrs []slog.Attr // attrs if non-empty
}

func (h *OLHandler) withGroupOrAttrs(goa groupOrAttrs) *OLHandler {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

func (h *OLHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

func (h *OLHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

// Rather than having a topLogger, we will have a topHandler that each subsystem will use to fetch its own logger with the appropriate level, since 
// each subsystem will have its own log level.
var TopLogger slog.Logger

func LoadLoggers() error {
	level, err := ParseLevelString(Conf.Trace.Cgroups)
	if (err != nil) {
		return err
	}
	// Open json log file as output instead of stdout.
	if Conf.Trace.Enable_JSON == true {
		olpath, err := GetOlPath(&cli.Context{})
		if err != nil {
			return err
		}
		logFilePath := path.Join(olpath, "worker.json")
		f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			panic(fmt.Errorf("Cannot open log file at %s", logFilePath))
		}
		TopLogger = *slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{Level: level}))

	// Using the OL's own handler.  
	} else {
		TopLogger = *slog.New(NewOLHandler(slog.LevelInfo, os.Stdout))
	}
	return nil
}

// Subsystem will be calling this function to get a copy of the topLogger with a specified level
func FetchLogger() error {
	
	return nil
}

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
		level.Set(slog.LevelError+1)
	} else {
		return level, fmt.Errorf("Unknown log level: %s", conf)
	}
	return level, nil
}


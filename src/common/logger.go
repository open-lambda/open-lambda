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

// The system's unique handler which format of the output can be freely modified to the owner's preference
type OLHandler struct {
	level   slog.Leveler
	goas 	[]groupOrAttrs
	mu 		*sync.Mutex
	out 	io.Writer
}

// Create a new OLHandler that implements the slog.Handler interface
func NewOLHandler(out io.Writer, level slog.Leveler) *OLHandler {
	if level == nil {
		// default to INFO (might not actually need this)
		level = slog.LevelInfo
	}
	return &OLHandler{level, nil, &sync.Mutex{}, out}
}

func (h *OLHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *OLHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)
	if !r.Time.IsZero() {
		buf = h.appendAttr(buf, slog.Time(slog.TimeKey, r.Time))
	}
	buf = h.appendAttr(buf, slog.Any("", r.Level))
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		buf = h.appendAttr(buf, slog.String(slog.SourceKey, fmt.Sprintf("%s:%d", f.File, f.Line)))
	}
	buf = h.appendAttr(buf, slog.String(slog.MessageKey, r.Message))
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
			buf = fmt.Appendf(buf, "%*s%s: ", "", goa.group)
		} else {
			for _, a := range goa.attrs {
				buf = h.appendAttr(buf, a)
			}
		}
	}
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a)
		return true
	})
	buf = append(buf, "\n"...)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

func (h *OLHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	a.Value = a.Value.Resolve()
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return buf
	}

	switch a.Value.Kind() {
	case slog.KindString:
		// Quote string values, to make them easy to parse.
		buf = fmt.Appendf(buf, "%s: %q ", a.Key, a.Value.String())
	case slog.KindTime:
		// Currently using a time format that matches the old log's format. Can change to any preference.
		buf = fmt.Appendf(buf, "%s ", a.Value.Time().Format("2006/01/02 15:04:05.999999"))
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
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga)
		}
	default:
		buf = fmt.Appendf(buf, "%s %s ", a.Key, a.Value)
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

// A LevelHandler wraps a slog.Handler with an Enabled method
// that returns false for levels below a minimum.
type LevelHandler struct {
	level   slog.Leveler
	handler slog.Handler
}

// NewLevelHandler returns a LevelHandler with the given level.
// All methods except Enabled delegate to h.
func NewLevelHandler(level slog.Leveler, h slog.Handler) *LevelHandler {
	// Optimization: avoid chains of LevelHandlers.
	if lh, ok := h.(*LevelHandler); ok {
		h = lh.Handler()
	}
	return &LevelHandler{level, h}
}

func (h *LevelHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *LevelHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.handler.Handle(ctx, r)
}

func (h *LevelHandler) Handler() slog.Handler {
	return h.handler
}

func (h *LevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewLevelHandler(h.level, h.handler.WithAttrs(attrs))
}

func (h *LevelHandler) WithGroup(name string) slog.Handler {
	return NewLevelHandler(h.level, h.handler.WithGroup(name))
}

// Rather than having a topLogger, we will have a topHandler that each subsystem will use to fetch its own logger with the appropriate level, 
// since each subsystem will have its own log level.
var TopHandler slog.Handler

func LoadLoggers() error {
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
		TopHandler = slog.NewJSONHandler(f, &slog.HandlerOptions{})

	// Using the OL's own handler.  
	} else {
		TopHandler = NewOLHandler(os.Stdout, slog.LevelInfo)
	}
	return nil
}

// Subsystem will be calling this function to get a copy of the topLogger with a specified level
func FetchLogger(ilevel string) (slog.Logger, error) {
	level, err := ParseLevelString(ilevel)
	if (err != nil) {
		return *slog.Default(), err
	}
	logger := slog.New(NewLevelHandler(level, TopHandler))
	return *logger, nil
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


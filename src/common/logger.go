package common

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/urfave/cli/v2"
)

// the current slog does not support enable/disable feature, might need a wrap-arround struct
var CgTopLogger slog.Logger

func LoadLoggers() error {
	if Conf.Trace.Enable_JSON == true {
		level := new(slog.LevelVar)
		if Conf.Trace.Cgroups == "INFO" {
			level.Set(slog.LevelInfo)
		} else if Conf.Trace.Cgroups == "WARN" {
			level.Set(slog.LevelWarn)
		} else if Conf.Trace.Cgroups == "ERROR" {
			level.Set(slog.LevelError)
		}
		olpath, err := GetOlPath(&cli.Context{})
		if err != nil {
			return err
		}
		logFilePath := path.Join(olpath, "worker.json")
		f, err := os.OpenFile(logFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			panic(fmt.Errorf("Cannot open log file at %s", logFilePath))
		}
		CgTopLogger = *slog.New(slog.NewJSONHandler(f, &slog.HandlerOptions{Level: level}))

	} else {
		// default slog's logger does not support level configuration at the moment 
		//(https://www.reddit.com/r/golang/comments/153svuq/slog_how_to_access_the_default_log_format/)
		// also it seems like the default logger's handler is not exported either
		//(https://groups.google.com/g/golang-nuts/c/aJPXT2NF-Lc)
		CgTopLogger = *slog.Default()
	}
	return nil
}
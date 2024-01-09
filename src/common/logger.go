package common

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/urfave/cli/v2"
)

// the current slog does not support enable/disable feature, might need a wrap-arround struct
var TopLogger slog.Logger

func LoadLoggers() error {
	if Conf.Trace.Enable_JSON == true {
		// Will we be using the same log level for all subsystems? If no, we need a seperate function for each subsystem OR a LoadLoggerHelper()
		// to avoid code repetition, but that will mean a different toplogger for each subsystem. (Slog API does not allow changing a logger's
		// level after initilization). However, we could set up the toplogger not at the start of the server but only when the subsystem fetch
		// its own logger, but that means the subsystems will
		// not actually be sharing the toplogger though. 
		level, err := ParseLevelString(Conf.Trace.Cgroups)
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

	} else {
		// If the default slog's logger should not be used, then should we use their provided text format as our default then?
		// We still need a format for outputing to the terminal though.
		// (The only three API provided are default, text, and json)
		TopLogger = *slog.Default()
	}
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
		level.Set(slog.LevelError+1)
	} else {
		return level, fmt.Errorf("Unknown log level: %s", conf)
	}
	return level, nil
}
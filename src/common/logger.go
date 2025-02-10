package common

import (
	"log/slog"
	"os"
)

// Default logger for the entire system (Any call from log will immediately be written with a JSON handler)
// TODO: Only format it as JSON for cgroups but any other systems calling log.Printf should just get a regular log output
var defaultSystemLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

// Set the defaultSystemLogger as default logger when common is initialized
func init() {
	slog.SetDefault(defaultSystemLogger)
}

// For cgroups, it should clone the default logger and append an additional attribute (i.e. name, id, etc.)
func LoadCgroupLogger(args ...any) *slog.Logger {
	// Clone the defaultCgroupLogger and add name as an additional attribute
	cgroupLogger := defaultSystemLogger.With(args)
	return cgroupLogger
}

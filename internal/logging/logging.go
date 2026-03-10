// Package logging initializes the default slog logger for the application.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Init configures the global slog default logger with the given level string.
// Valid values: "DEBUG", "INFO", "WARN"/"WARNING", "ERROR". Defaults to INFO.
func Init(level string) {
	var lvl slog.Level
	switch strings.ToUpper(level) {
	case "DEBUG":
		lvl = slog.LevelDebug
	case "WARN", "WARNING":
		lvl = slog.LevelWarn
	case "ERROR":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(h))
}

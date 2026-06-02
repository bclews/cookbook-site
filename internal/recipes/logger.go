package recipes

import (
	"log/slog"
	"os"
)

// Logger is the package-level logger for structured logging.
// It can be replaced with a custom logger using SetLogger.
var Logger *slog.Logger

// logLevel allows dynamic level adjustment without recreating the logger.
var logLevel = new(slog.LevelVar)

func init() {
	// Default to a text handler writing to stderr
	logLevel.Set(slog.LevelInfo)
	Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
}

// SetLogger allows replacing the package logger with a custom one.
func SetLogger(logger *slog.Logger) {
	if logger != nil {
		Logger = logger
	}
}

// SetLogLevel sets the logging level dynamically without recreating the logger.
func SetLogLevel(level slog.Level) {
	logLevel.Set(level)
}

package recipes

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestSetLogger(t *testing.T) {
	// Save original logger to restore after test
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	t.Run("replaces logger with custom logger", func(t *testing.T) {
		var buf bytes.Buffer
		customLogger := slog.New(slog.NewTextHandler(&buf, nil))

		SetLogger(customLogger)

		if Logger != customLogger {
			t.Error("SetLogger did not replace the logger")
		}

		// Verify the new logger works
		Logger.Info("test message")
		if !strings.Contains(buf.String(), "test message") {
			t.Errorf("Custom logger should have logged message, got: %s", buf.String())
		}
	})

	t.Run("ignores nil logger", func(t *testing.T) {
		// Set a known logger first
		var buf bytes.Buffer
		knownLogger := slog.New(slog.NewTextHandler(&buf, nil))
		SetLogger(knownLogger)

		// Try to set nil
		SetLogger(nil)

		// Logger should still be the known logger
		if Logger != knownLogger {
			t.Error("SetLogger should ignore nil logger")
		}
	})
}

func TestSetLogLevel(t *testing.T) {
	// Save original logger to restore after test
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	t.Run("sets debug level", func(t *testing.T) {
		var buf bytes.Buffer
		// Create a new logger with our logLevel variable
		SetLogLevel(slog.LevelDebug)
		customLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level: logLevel,
		}))
		SetLogger(customLogger)

		Logger.Debug("debug message")
		if !strings.Contains(buf.String(), "debug message") {
			t.Errorf("Debug message should be logged at debug level, got: %s", buf.String())
		}
	})

	t.Run("sets warn level", func(t *testing.T) {
		var buf bytes.Buffer
		SetLogLevel(slog.LevelWarn)
		customLogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
			Level: logLevel,
		}))
		SetLogger(customLogger)

		Logger.Info("info message")
		if strings.Contains(buf.String(), "info message") {
			t.Error("Info message should not be logged at warn level")
		}

		Logger.Warn("warn message")
		if !strings.Contains(buf.String(), "warn message") {
			t.Errorf("Warn message should be logged at warn level, got: %s", buf.String())
		}
	})
}

func TestDefaultLogger(t *testing.T) {
	// The default logger should be set by init()
	if Logger == nil {
		t.Error("Default logger should be initialized")
	}
}

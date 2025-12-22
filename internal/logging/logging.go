package logging

// Package logging provides structured logger setup and helpers.

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// Logger wraps zerolog.Logger to provide a clean interface for structured logging.
// All browser automation actions should use this logger for observability.
type Logger struct {
	logger zerolog.Logger
}

// NewLogger creates a new Logger instance based on the provided configuration.
// It sets up output format (JSON or console), log level, and timestamp formatting.
func NewLogger(logLevel string, logFormat string) *Logger {
	// Set global log level based on config
	var level zerolog.Level
	switch logLevel {
	case "debug":
		level = zerolog.DebugLevel
	case "info":
		level = zerolog.InfoLevel
	case "warn":
		level = zerolog.WarnLevel
	case "error":
		level = zerolog.ErrorLevel
	default:
		level = zerolog.InfoLevel // Default fallback
	}
	zerolog.SetGlobalLevel(level)

	// Configure output writer based on format
	var outputWriter io.Writer

	if logFormat == "console" {
		// Human-readable console output with colors
		outputWriter = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05",
			NoColor:    false,
		}
	} else {
		// JSON format - use os.Stdout directly (zerolog will format as JSON)
		outputWriter = os.Stdout
	}

	// Create logger with context
	logger := zerolog.New(outputWriter).With().
		Timestamp().
		Logger()

	return &Logger{
		logger: logger,
	}
}

// Debug creates a debug-level log event.
// Usage: logger.Debug().Str("key", "value").Msg("message")
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Info creates an info-level log event.
// Usage: logger.Info().Str("key", "value").Msg("message")
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Warn creates a warn-level log event.
// Usage: logger.Warn().Str("key", "value").Msg("message")
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// Error creates an error-level log event.
// Usage: logger.Error().Err(err).Msg("message")
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// WithComponent creates a new Logger instance with a component field pre-set.
// This is useful for tagging all logs from a specific module (e.g., "auth", "stealth").
// Usage: componentLogger := logger.WithComponent("auth"); componentLogger.Info().Msg("Login started")
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		logger: l.logger.With().Str("component", component).Logger(),
	}
}

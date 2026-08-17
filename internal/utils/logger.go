package utils

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"
)

// Logger is a global logger for backward compatibility with existing code
// This is initialized with default settings and can be overridden
var Logger *slog.Logger

func init() {
	// Initialize with default settings (will be overridden by config)
	Logger = NewLogger(LoggerConfig{
		Level:       "info",
		Format:      "json",
		ServiceName: "trade-detection",
	})
}

// LoggerConfig for structured logger configuration
type LoggerConfig struct {
	Level       string
	Format      string // "json" or "text"
	AddSource   bool
	Output      io.Writer
	ServiceName string // For structured context
}

// NewLogger creates a configured structured logger
func NewLogger(cfg LoggerConfig) *slog.Logger {
	level := parseLogLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var handler slog.Handler
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	if cfg.Format == "text" {
		handler = slog.NewTextHandler(output, opts)
	} else {
		handler = slog.NewJSONHandler(output, opts)
	}

	logger := slog.New(handler)

	// Add service context if provided
	if cfg.ServiceName != "" {
		logger = logger.With(
			"service", cfg.ServiceName,
			"hostname", getHostname(),
			"pid", os.Getpid(),
		)
	}

	return logger
}

// WithContext adds context values to logger for request tracking
func WithContext(logger *slog.Logger, ctx context.Context) *slog.Logger {
	// Extract request_id from context
	if reqID := ctx.Value("request_id"); reqID != nil {
		logger = logger.With("request_id", reqID)
	}
	// Extract user_id from context
	if userID := ctx.Value("user_id"); userID != nil {
		logger = logger.With("user_id", userID)
	}
	// Extract trace_id from context for distributed tracing
	if traceID := ctx.Value("trace_id"); traceID != nil {
		logger = logger.With("trace_id", traceID)
	}
	return logger
}

// WithFields adds structured fields to logger
func WithFields(logger *slog.Logger, fields map[string]interface{}) *slog.Logger {
	args := make([]interface{}, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}
	return logger.With(args...)
}

// LogSlowOperation logs operations that exceed threshold
// Usage: defer LogSlowOperation(logger, "database_query", 100*time.Millisecond)()
func LogSlowOperation(logger *slog.Logger, operation string, threshold time.Duration) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		if duration > threshold {
			logger.Warn("Slow operation detected",
				"operation", operation,
				"duration_ms", duration.Milliseconds(),
				"threshold_ms", threshold.Milliseconds(),
			)
		} else {
			logger.Debug("Operation completed",
				"operation", operation,
				"duration_ms", duration.Milliseconds(),
			)
		}
	}
}

// Fatal logs error and exits with code 1
func Fatal(logger *slog.Logger, msg string, args ...any) {
	logger.Error(msg, args...)
	os.Exit(1)
}

// parseLogLevel converts string to slog.Level
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// getHostname returns the system hostname
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

package logging

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"fulcrumproject.org/core/internal/config"
	"github.com/go-chi/chi/v5/middleware"
	slogGorm "github.com/orandin/slog-gorm"
	gormLogger "gorm.io/gorm/logger"
)

// NewGormLogger configures the logger based on the log format and level from config
func NewGormLogger(cfg *config.DBConfig) gormLogger.Interface {
	var handler slog.Handler

	// Get log level from config
	level := cfg.GetLogLevel()

	// Configure the options with the log level
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Configure the handler based on format
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slogGorm.New(
		slogGorm.WithHandler(handler),
		slogGorm.WithTraceAll(),
	)
}

// NewLogger configures the logger based on the log format and level from config
func NewLogger(cfg *config.Config) *slog.Logger {
	var handler slog.Handler

	// Get log level from config
	level := cfg.LogConfig.GetLogLevel()

	// Configure the options with the log level
	opts := &slog.HandlerOptions{
		Level: level,
	}

	// Configure the handler based on format
	if cfg.LogConfig.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// SlogFormatter is a custom log formatter for chi that uses slog
type SlogFormatter struct {
	Logger *slog.Logger
}

// NewLogEntry creates a new log entry for an HTTP request
func (sf *SlogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &SlogLogEntry{
		Logger: sf.Logger,
		req:    r,
	}
}

// Panic logs the panic details using slog
func (l *SlogLogEntry) Panic(v interface{}, stack []byte) {
	l.Logger.Error("HTTP Request Panic",
		"method", l.req.Method, "uri", l.req.RequestURI, "panic", v, "stack", string(stack))
}

// SlogLogEntry is a log entry that uses slog
type SlogLogEntry struct {
	Logger *slog.Logger
	req    *http.Request
}

// Write logs the response details using slog
func (l *SlogLogEntry) Write(status, bytes int, header http.Header, elapsed time.Duration, extra interface{}) {
	l.Logger.Info("HTTP Request",
		"method", l.req.Method, "uri", l.req.RequestURI, "status", status,
		"bytes", bytes, "elapsed", elapsed.String(), "remote", l.req.RemoteAddr)
}

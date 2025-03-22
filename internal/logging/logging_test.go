package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"fulcrumproject.org/core/internal/config"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestOutput captures stdout for testing logger output
func setupTestOutput() (*os.File, *os.File, io.Reader) {
	// Save the original stdout
	oldStdout := os.Stdout

	// Create a pipe to replace stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Return original stdout, pipe writer, and pipe reader
	return oldStdout, w, r
}

// restoreOutput restores the original stdout and captures the test output
func restoreOutput(oldStdout *os.File, w *os.File, r io.Reader) string {
	// Flush the writer and close it
	w.Close()

	// Restore the original stdout
	os.Stdout = oldStdout

	// Read from the pipe into a buffer
	buf := new(bytes.Buffer)
	io.Copy(buf, r)
	return buf.String()
}

// TestNewGormLogger tests the NewGormLogger function with different configurations
func TestNewGormLogger(t *testing.T) {
	testCases := []struct {
		name           string
		config         *config.DBConfig
		expectedFormat string
		expectedLevel  slog.Level
	}{
		{
			name: "JSON logger with error level",
			config: &config.DBConfig{
				LogFormat: "json",
				LogLevel:  "error",
			},
			expectedFormat: "json",
			expectedLevel:  slog.LevelError,
		},
		{
			name: "Text logger with info level",
			config: &config.DBConfig{
				LogFormat: "text",
				LogLevel:  "info",
			},
			expectedFormat: "text",
			expectedLevel:  slog.LevelInfo,
		},
		{
			name: "Default logger (when format not specified)",
			config: &config.DBConfig{
				LogLevel: "warn",
			},
			expectedFormat: "text", // Default to text
			expectedLevel:  slog.LevelWarn,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the logger
			logger := NewGormLogger(tc.config)

			// Verify it's not nil (basic creation test)
			assert.NotNil(t, logger)

			// Additional checks could be added here, but would require exposing
			// internal fields of the slogGorm implementation
		})
	}
}

// TestNewLogger tests the NewLogger function with different configurations
func TestNewLogger(t *testing.T) {
	testCases := []struct {
		name           string
		config         *config.Config
		expectedFormat string
		expectedLevel  slog.Level
		testLog        string
		shouldContain  string
		shouldNotLog   bool
	}{
		{
			name: "JSON logger with error level",
			config: &config.Config{
				LogConfig: config.LogConfig{
					Format: "json",
					Level:  "error",
				},
			},
			expectedFormat: "json",
			expectedLevel:  slog.LevelError,
			testLog:        "error test message",
			shouldContain:  `"level":"ERROR"`,
		},
		{
			name: "Text logger with info level",
			config: &config.Config{
				LogConfig: config.LogConfig{
					Format: "text",
					Level:  "info",
				},
			},
			expectedFormat: "text",
			expectedLevel:  slog.LevelInfo,
			testLog:        "info test message",
			shouldContain:  "INFO",
		},
		{
			name: "Warn level logger should not log info",
			config: &config.Config{
				LogConfig: config.LogConfig{
					Format: "text",
					Level:  "warn",
				},
			},
			expectedFormat: "text",
			expectedLevel:  slog.LevelWarn,
			testLog:        "this info should not appear",
			shouldNotLog:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture stdout
			oldStdout, w, r := setupTestOutput()

			// Create the logger
			logger := NewLogger(tc.config)

			// Test basic properties
			assert.NotNil(t, logger)

			// Test actual logging
			if tc.shouldNotLog {
				logger.Info(tc.testLog)
			} else {
				// If we're expecting INFO level, log at INFO level, otherwise use ERROR
				if tc.expectedLevel == slog.LevelInfo {
					logger.Info(tc.testLog)
				} else {
					logger.Error(tc.testLog)
				}
			}

			// Restore stdout and get output
			output := restoreOutput(oldStdout, w, r)

			// Check output
			if tc.shouldNotLog {
				assert.Empty(t, output, "Expected no log output for level filtering")
			} else {
				assert.Contains(t, output, tc.shouldContain, "Log output should contain expected format elements")
				assert.Contains(t, output, tc.testLog, "Log output should contain the test message")
			}
		})
	}
}

// TestSlogFormatter tests the SlogFormatter implementation
func TestSlogFormatter(t *testing.T) {
	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create formatter
	formatter := &SlogFormatter{
		Logger: logger,
	}

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	// Create a log entry
	entry := formatter.NewLogEntry(req)

	// Verify entry is of the correct type
	logEntry, ok := entry.(*SlogLogEntry)
	require.True(t, ok, "Log entry should be of type *SlogLogEntry")
	assert.Equal(t, logger, logEntry.Logger)
	assert.Equal(t, req, logEntry.req)
}

// TestSlogLogEntryWrite tests the Write method of SlogLogEntry
func TestSlogLogEntryWrite(t *testing.T) {
	// Capture stdout
	oldStdout, w, r := setupTestOutput()

	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	// Create a log entry
	logEntry := &SlogLogEntry{
		Logger: logger,
		req:    req,
	}

	// Call the Write method
	status := 200
	bytesWritten := 100
	elapsed := 50 * time.Millisecond
	header := http.Header{}
	logEntry.Write(status, bytesWritten, header, elapsed, nil)

	// Restore stdout and get output
	output := restoreOutput(oldStdout, w, r)

	// Check output
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "HTTP Request")
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/test")
	assert.Contains(t, output, "200")
	assert.Contains(t, output, "127.0.0.1:1234")
}

// TestSlogLogEntryPanic tests the Panic method of SlogLogEntry
func TestSlogLogEntryPanic(t *testing.T) {
	// Capture stdout
	oldStdout, w, r := setupTestOutput()

	// Create a logger for testing
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	// Create a test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	// Create a log entry
	logEntry := &SlogLogEntry{
		Logger: logger,
		req:    req,
	}

	// Call the Panic method
	panicValue := "test panic"
	stackTrace := []byte("fake stack trace")
	logEntry.Panic(panicValue, stackTrace)

	// Restore stdout and get output
	output := restoreOutput(oldStdout, w, r)

	// Check output
	assert.Contains(t, output, "ERROR")
	assert.Contains(t, output, "HTTP Request Panic")
	assert.Contains(t, output, panicValue)
	assert.Contains(t, output, "fake stack trace")
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/test")
}

// TestIntegrationWithChiMiddleware tests the integration with Chi middleware
func TestIntegrationWithChiMiddleware(t *testing.T) {
	// Capture stdout
	oldStdout, w, r := setupTestOutput()

	// Create a logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create a request logger middleware
	requestLogger := middleware.RequestLogger(&SlogFormatter{Logger: logger})

	// Create a simple handler
	handler := requestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	// Create a request and response recorder
	req := httptest.NewRequest("GET", "/test-middleware", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()

	// Handle the request
	handler.ServeHTTP(rec, req)

	// Restore stdout and get output
	output := restoreOutput(oldStdout, w, r)

	// Check response
	assert.Equal(t, http.StatusOK, rec.Code)

	// Check log output
	assert.Contains(t, output, "INFO")
	assert.Contains(t, output, "HTTP Request")
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/test-middleware")
	assert.Contains(t, output, "200")
}

// TestLoggerWithContext tests using the logger with context
func TestLoggerWithContext(t *testing.T) {
	// Capture stdout
	oldStdout, w, r := setupTestOutput()

	// Create a logger
	logger := NewLogger(&config.Config{
		LogConfig: config.LogConfig{
			Format: "json",
			Level:  "info",
		},
	})

	// Create a context with values
	ctx := context.Background()
	logger = logger.With("requestID", "12345")

	// Log with context values
	logger.InfoContext(ctx, "context test message", "user", "test-user")

	// Restore stdout and get output
	output := restoreOutput(oldStdout, w, r)

	// Parse JSON output
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &logEntry)
	require.NoError(t, err, "Should be able to parse JSON log output")

	// Check fields
	assert.Equal(t, "INFO", logEntry["level"])
	assert.Equal(t, "context test message", logEntry["msg"])
	assert.Equal(t, "12345", logEntry["requestID"])
	assert.Equal(t, "test-user", logEntry["user"])
}

package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func TestNew_JSONFormat(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	// Temporarily redirect stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	logger, err := New(Config{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Capture the output
	logger.Info().Msg("test message")
	w.Close()
	buf.ReadFrom(r)

	// Verify JSON format
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}

	if logEntry["message"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", logEntry["message"])
	}
}

func TestNew_TextFormat(t *testing.T) {
	_, err := New(Config{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Just verify we can create the logger without errors
	// Text format uses ConsoleWriter which is hard to test output directly
}

func TestNew_LogLevels(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected zerolog.Level
	}{
		{
			name:     "debug level",
			level:    "debug",
			expected: zerolog.DebugLevel,
		},
		{
			name:     "info level",
			level:    "info",
			expected: zerolog.InfoLevel,
		},
		{
			name:     "warn level",
			level:    "warn",
			expected: zerolog.WarnLevel,
		},
		{
			name:     "error level",
			level:    "error",
			expected: zerolog.ErrorLevel,
		},
		{
			name:     "invalid level defaults to info",
			level:    "invalid",
			expected: zerolog.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(Config{
				Level:  tt.level,
				Format: "json",
				Output: "stdout",
			})
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			if zerolog.GlobalLevel() != tt.expected {
				t.Errorf("Expected global log level %v, got %v", tt.expected, zerolog.GlobalLevel())
			}

			// Reset global level
			zerolog.SetGlobalLevel(zerolog.InfoLevel)
		})
	}
}

func TestNew_FileOutput(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	logger, err := New(Config{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Write a log entry
	logger.Info().Msg("test file output")

	// Verify file was created and contains the log
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test file output") {
		t.Errorf("Log file does not contain expected message")
	}
}

func TestNew_FileOutput_DefaultPath(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)

	// Create temp directory and change to it
	tempDir := t.TempDir()
	os.Chdir(tempDir)

	logger, err := New(Config{
		Level:  "info",
		Format: "json",
		Output: "file",
		// No FilePath specified - should use default
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Write a log entry
	logger.Info().Msg("test default file path")

	// Verify default log file was created
	defaultPath := filepath.Join(tempDir, "verus-gateway.log")
	if _, err := os.Stat(defaultPath); os.IsNotExist(err) {
		t.Errorf("Default log file was not created at %s", defaultPath)
	}
}

func TestNew_FileOutput_InvalidPath(t *testing.T) {
	// Try to create log file in non-existent directory
	_, err := New(Config{
		Level:    "info",
		Format:   "json",
		Output:   "file",
		FilePath: "/nonexistent/directory/test.log",
	})
	if err == nil {
		t.Error("Expected error for invalid file path, got nil")
	}
}

func TestNew_StderrOutput(t *testing.T) {
	_, err := New(Config{
		Level:  "info",
		Format: "json",
		Output: "stderr",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Just verify we can create the logger without errors
}

func TestFromContext_WithLogger(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Str("test", "value").Logger()
	ctx := context.WithValue(context.Background(), LoggerKey, &logger)

	retrievedLogger := FromContext(ctx)
	if retrievedLogger == nil {
		t.Fatal("FromContext() returned nil")
	}

	// Verify it's the same logger (has the same context fields)
	// We can't directly compare loggers, but we can verify they have the same context
	if retrievedLogger != &logger {
		t.Error("FromContext() did not return the same logger instance")
	}
}

func TestFromContext_WithoutLogger(t *testing.T) {
	ctx := context.Background()

	retrievedLogger := FromContext(ctx)
	if retrievedLogger == nil {
		t.Fatal("FromContext() returned nil for context without logger")
	}

	// Should return a default logger
	// Verify it's usable
	retrievedLogger.Info().Msg("test")
}

func TestWithContext(t *testing.T) {
	logger := zerolog.New(os.Stdout).With().Str("test", "value").Logger()
	ctx := context.Background()

	newCtx := WithContext(ctx, &logger)

	// Verify logger was added to context
	retrievedLogger := FromContext(newCtx)
	if retrievedLogger == nil {
		t.Fatal("Logger was not added to context")
	}

	if retrievedLogger != &logger {
		t.Error("Retrieved logger is not the same as the one added")
	}
}

func TestAddRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	loggerWithID := AddRequestID(logger, "test-request-id")
	loggerWithID.Info().Msg("test")

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["request_id"] != "test-request-id" {
		t.Errorf("Expected request_id 'test-request-id', got %v", logEntry["request_id"])
	}
}

func TestAddChain(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	loggerWithChain := AddChain(logger, "vrsctest")
	loggerWithChain.Info().Msg("test")

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["chain"] != "vrsctest" {
		t.Errorf("Expected chain 'vrsctest', got %v", logEntry["chain"])
	}
}

func TestAddTXID(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	loggerWithTXID := AddTXID(logger, "abc123")
	loggerWithTXID.Info().Msg("test")

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["txid"] != "abc123" {
		t.Errorf("Expected txid 'abc123', got %v", logEntry["txid"])
	}
}

func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short string (4 chars)",
			input:    "test",
			expected: "****",
		},
		{
			name:     "short string (3 chars)",
			input:    "abc",
			expected: "****",
		},
		{
			name:     "short string (1 char)",
			input:    "a",
			expected: "****",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "****",
		},
		{
			name:     "long string",
			input:    "password123",
			expected: "pa****23",
		},
		{
			name:     "5 char string",
			input:    "12345",
			expected: "12****45",
		},
		{
			name:     "very long string",
			input:    "supersecretpassword",
			expected: "su****rd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitiveData(tt.input)
			if result != tt.expected {
				t.Errorf("MaskSensitiveData(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNew_TextFormatNoColor(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test-nocolor.log")

	logger, err := New(Config{
		Level:    "info",
		Format:   "text",
		Output:   "file",
		FilePath: logFile,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Write a log entry
	logger.Info().Msg("test no color output")

	// Verify file was created and doesn't contain ANSI color codes
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	// ANSI color codes start with \x1b or \033
	if strings.Contains(string(content), "\x1b") || strings.Contains(string(content), "\033") {
		t.Errorf("Log file contains ANSI color codes when NoColor should be true")
	}

	if !strings.Contains(string(content), "test no color output") {
		t.Errorf("Log file does not contain expected message")
	}
}

func TestLogger_ChainedContext(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	// Chain multiple context additions
	enrichedLogger := AddRequestID(logger, "req-123")
	enrichedLogger = AddChain(enrichedLogger, "vrsctest")
	enrichedLogger = AddTXID(enrichedLogger, "tx-456")

	enrichedLogger.Info().Msg("test chained context")

	// Parse JSON output
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if logEntry["request_id"] != "req-123" {
		t.Errorf("Expected request_id 'req-123', got %v", logEntry["request_id"])
	}
	if logEntry["chain"] != "vrsctest" {
		t.Errorf("Expected chain 'vrsctest', got %v", logEntry["chain"])
	}
	if logEntry["txid"] != "tx-456" {
		t.Errorf("Expected txid 'tx-456', got %v", logEntry["txid"])
	}
}

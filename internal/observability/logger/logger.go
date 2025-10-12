package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// contextKey is the type for context keys
type contextKey string

const (
	// LoggerKey is the context key for the logger
	LoggerKey contextKey = "logger"
)

// Config holds logger configuration
type Config struct {
	Level    string // debug, info, warn, error
	Format   string // json, text
	Output   string // stdout, stderr, file
	FilePath string
}

// New creates a new zerolog logger
func New(cfg Config) (zerolog.Logger, error) {
	// Set log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Set output
	var output io.Writer
	switch cfg.Output {
	case "stderr":
		output = os.Stderr
	case "file":
		if cfg.FilePath == "" {
			cfg.FilePath = "verus-gateway.log"
		}
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return zerolog.Logger{}, err
		}
		output = file
	default:
		output = os.Stdout
	}

	// Set format
	if cfg.Format == "text" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    cfg.Output == "file",
		}
	}

	// Create logger
	logger := zerolog.New(output).With().
		Timestamp().
		Caller().
		Logger()

	return logger, nil
}

// FromContext retrieves the logger from context
func FromContext(ctx context.Context) *zerolog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*zerolog.Logger); ok {
		return logger
	}
	// Return a default logger if not found
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	return &logger
}

// WithContext adds a logger to the context
func WithContext(ctx context.Context, logger *zerolog.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// AddRequestID adds a request ID to the logger
func AddRequestID(logger zerolog.Logger, requestID string) zerolog.Logger {
	return logger.With().Str("request_id", requestID).Logger()
}

// AddChain adds a chain ID to the logger
func AddChain(logger zerolog.Logger, chainID string) zerolog.Logger {
	return logger.With().Str("chain", chainID).Logger()
}

// AddTXID adds a transaction ID to the logger
func AddTXID(logger zerolog.Logger, txid string) zerolog.Logger {
	return logger.With().Str("txid", txid).Logger()
}

// MaskSensitiveData masks sensitive data like passwords
func MaskSensitiveData(data string) string {
	if len(data) <= 4 {
		return "****"
	}
	return data[:2] + "****" + data[len(data)-2:]
}

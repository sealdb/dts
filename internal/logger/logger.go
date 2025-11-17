package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pg/dts/internal/config"
	"github.com/sirupsen/logrus"
)

var (
	// Logger is the global logger instance
	Logger *logrus.Logger
)

// Init initializes the logger
func Init(cfg *config.LogConfig) error {
	Logger = logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	Logger.SetLevel(level)

	// Set log format
	switch cfg.Format {
	case "json":
		Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	case "text":
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	default:
		return fmt.Errorf("invalid log format: %s (supported: json, text)", cfg.Format)
	}

	// Set log output
	var output io.Writer
	switch cfg.Output {
	case "stdout":
		output = os.Stdout
	case "stderr":
		output = os.Stderr
	case "":
		output = os.Stdout
	default:
		// File path
		dir := filepath.Dir(cfg.Output)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		output = file
	}

	Logger.SetOutput(output)

	return nil
}

// GetLogger returns the logger instance
func GetLogger() *logrus.Logger {
	if Logger == nil {
		// If not initialized, use default configuration
		Logger = logrus.New()
		Logger.SetLevel(logrus.InfoLevel)
		Logger.SetFormatter(&logrus.JSONFormatter{})
		Logger.SetOutput(os.Stdout)
	}
	return Logger
}

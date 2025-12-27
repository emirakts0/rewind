package logging

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

var logFile *lumberjack.Logger

// Setup initializes the logging system
func Setup(logPath string, debug bool) error {
	// Ensure log directory exists
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create rotating log file
	logFile = &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     7,     // days
		Compress:   false, // Don't compress to make debugging easier
	}

	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	// Create multi-writer for both file and stdout (if available)
	var writers []io.Writer
	writers = append(writers, logFile)

	// Try to add stdout if available (not available in GUI mode on Windows)
	if fileInfo, _ := os.Stdout.Stat(); fileInfo != nil {
		writers = append(writers, os.Stdout)
	}

	multiWriter := io.MultiWriter(writers...)

	// Setup slog
	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))

	// Also redirect standard log package to the file
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	slog.Info("logging initialized", "path", logPath, "debug", debug)

	// Write a test message directly to file to ensure it works
	fmt.Fprintf(logFile, "=== Rewind Log Started ===\n")

	return nil
}

// Close closes the log file
func Close() {
	if logFile != nil {
		slog.Info("logging shutdown")
		logFile.Close()
	}
}

// GetLogPath returns the log file path relative to executable
func GetLogPath() string {
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to current directory
		return filepath.Join(".", "logs", "rewind.log")
	}
	exeDir := filepath.Dir(exePath)
	return filepath.Join(exeDir, "logs", "rewind.log")
}

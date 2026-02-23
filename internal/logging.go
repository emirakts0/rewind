package internal

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

func SetDefaultLogging() {
	logsDir, err := GetLogsDir()
	if err != nil {
		log.Printf("Failed to get logs directory: %v", err)
		logsDir = "."
	}

	logPath := filepath.Join(logsDir, "rewind.log")
	if err := SetupLogging(logPath, true); err != nil {
		log.Printf("Failed to setup logging: %v", err)
	}
}

// SetupLogging Setup initializes the logging system
func SetupLogging(logPath string, debug bool) error {
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

	var writers []io.Writer
	writers = append(writers, logFile)

	if fileInfo, _ := os.Stdout.Stat(); fileInfo != nil {
		writers = append(writers, os.Stdout)
	}

	multiWriter := io.MultiWriter(writers...)

	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(multiWriter, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))

	// Also redirect standard log package to the file
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	slog.Info("logging initialized", "path", logPath, "debug", debug)

	return nil
}

func CloseLogFile() {
	if logFile != nil {
		slog.Info("logging shutdown")
		err := logFile.Close()
		if err != nil {
			// todo
			slog.Error("failed to close log file", "error", err)
			return
		}
	}
}

package core

import (
	"fmt"
	"os"
	"time"
)

// Logger provides functionality for auditable logging.
type Logger struct {
	logFile *os.File
}

// NewLogger initializes a new Logger instance.
func NewLogger(filePath string) (*Logger, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	return &Logger{logFile: file}, nil
}

// LogMessage writes a log entry for a processed message.
func (l *Logger) LogMessage(messageID string, messageType string, details string) error {
	timestamp := time.Now().Format(time.RFC3339)
	logEntry := fmt.Sprintf("%s | ID: %s | Type: %s | Details: %s\n", timestamp, messageID, messageType, details)
	if _, err := l.logFile.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}
	return nil
}

// Close closes the log file.
func (l *Logger) Close() error {
	return l.logFile.Close()
}

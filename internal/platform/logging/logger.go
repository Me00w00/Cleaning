package logging

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type Logger struct {
	*slog.Logger
	file *os.File
}

func New(logFilePath string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0o755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	handler := slog.NewTextHandler(file, &slog.HandlerOptions{Level: slog.LevelInfo})

	return &Logger{
		Logger: slog.New(handler),
		file:   file,
	}, nil
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}

	return l.file.Close()
}

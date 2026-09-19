package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
)

// New creates a JSON slog logger writing to stdout, and optionally appending to filePath.
// If filePath is empty, only stdout is used.
func New(filePath string) (*slog.Logger, io.Closer, error) {
	writers := []io.Writer{os.Stdout}
	var file *os.File

	if filePath != "" {
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file: %w", err)
		}
		file = f
		writers = append(writers, f)
	}

	w := io.MultiWriter(writers...)
	log := slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if file != nil {
		return log, file, nil
	}
	return log, nopCloser{}, nil
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }

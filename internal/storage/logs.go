package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Log struct {
	file *os.File
	mu   sync.Mutex
}

type LogEvent struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	RunID     string `json:"run_id,omitempty"`
	PostID    string `json:"post_id,omitempty"`
	AssetID   string `json:"asset_id,omitempty"`
	AttemptID string `json:"attempt_id,omitempty"`
	Event     string `json:"event"`
	Message   string `json:"message,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
	Retryable bool   `json:"retryable,omitempty"`
}

func OpenLog(path string) (*Log, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Log{file: file}, nil
}

func (l *Log) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Log) Info(event LogEvent) error {
	return l.write("info", event)
}

func (l *Log) Error(event LogEvent) error {
	return l.write("error", event)
}

func (l *Log) write(level string, event LogEvent) error {
	if l == nil || l.file == nil {
		return nil
	}
	event.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	event.Level = level
	event.Message = RedactSecrets(event.Message)
	event.ErrorCode = RedactSecrets(event.ErrorCode)
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func RedactSecrets(value string) string {
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{"authorization", "bearer ", "cookie", "token", "secret"} {
		if strings.Contains(lower, marker) {
			return "[redacted]"
		}
	}
	return value
}

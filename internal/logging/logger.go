package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"system-sentinel/internal/metrics"
)

type Logger struct {
	logDir string
	mu     sync.Mutex
	file   *os.File
	date   string
}

type LogEntry struct {
	Timestamp string                  `json:"timestamp"`
	Type      string                  `json:"type"`
	Metric    string                  `json:"metric"`
	Reasons   []string                `json:"reasons,omitempty"`
	Metrics   metrics.MetricsSnapshot `json:"metrics"`
}

func NewLogger(logDir string) (*Logger, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log dir: %w", err)
	}

	logger := &Logger{logDir: logDir}
	if err := logger.rotateIfNeeded(); err != nil {
		return nil, err
	}

	return logger, nil
}

func (l *Logger) LogSample(snap metrics.MetricsSnapshot) error {
	return l.log("sample", "sample", nil, snap)
}

func (l *Logger) LogSpike(snap metrics.MetricsSnapshot, spikeTypes []string) error {
	metric := "multi"
	if len(spikeTypes) == 1 {
		metric = spikeTypes[0]
	}
	return l.log("spike", metric, spikeTypes, snap)
}

func (l *Logger) LogAlert(snap metrics.MetricsSnapshot, alertTypes []string) error {
	metric := "multi"
	if len(alertTypes) == 1 {
		metric = alertTypes[0]
	}
	return l.log("alert", metric, alertTypes, snap)
}

func (l *Logger) log(eventType, metric string, reasons []string, snap metrics.MetricsSnapshot) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.rotateIfNeeded(); err != nil {
		return err
	}

	entry := LogEntry{
		Timestamp: snap.Timestamp.Format(time.RFC3339),
		Type:      eventType,
		Metric:    metric,
		Reasons:   reasons,
		Metrics:   snap,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	if _, err := l.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write log: %w", err)
	}

	return nil
}

func (l *Logger) rotateIfNeeded() error {
	now := time.Now().UTC()
	currentDate := now.Format("2006-01-02")

	if l.file != nil && l.date == currentDate {
		return nil
	}

	if l.file != nil {
		l.file.Close()
	}

	filename := filepath.Join(l.logDir, fmt.Sprintf("metrics-%s.ndjson", currentDate))
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.file = file
	l.date = currentDate
	return nil
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

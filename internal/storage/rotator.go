package storage

import (
	"os"
	"path/filepath"
	"regexp"
	"time"
)

type Rotator struct {
	logDir        string
	retentionDays int
	stopCh        chan struct{}
	doneCh        chan struct{}
}

var filenamePattern = regexp.MustCompile(`^metrics-(\d{4}-\d{2}-\d{2})\.ndjson$`)

func NewRotator(logDir string, retentionDays int) *Rotator {
	return &Rotator{
		logDir:        logDir,
		retentionDays: retentionDays,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
}

func (r *Rotator) Start() {
	go r.run()
}

func (r *Rotator) Stop() {
	close(r.stopCh)
	<-r.doneCh
}

func (r *Rotator) run() {
	defer close(r.doneCh)

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	r.rotate()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.rotate()
		}
	}
}

func (r *Rotator) rotate() {
	entries, err := os.ReadDir(r.logDir)
	if err != nil {
		return
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -r.retentionDays)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		matches := filenamePattern.FindStringSubmatch(entry.Name())
		if len(matches) != 2 {
			continue
		}

		fileDate, err := time.Parse("2006-01-02", matches[1])
		if err != nil {
			continue
		}

		if fileDate.Before(cutoff) {
			filePath := filepath.Join(r.logDir, entry.Name())
			os.Remove(filePath)
		}
	}
}

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"system-sentinel/internal/alerts"
	"system-sentinel/internal/config"
	"system-sentinel/internal/logging"
	"system-sentinel/internal/metrics"
	"system-sentinel/internal/scripts"
	"system-sentinel/internal/spikes"
	"system-sentinel/internal/storage"
)

const version = "1.0.0"

func main() {
	var (
		configPath  string
		showVersion bool
	)

	flag.StringVar(&configPath, "config", "/etc/system-sentinel/config.yaml", "path to config.yaml")
	flag.BoolVar(&showVersion, "version", false, "show version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("system-sentinel v%s\n", version)
		os.Exit(0)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	collector := metrics.NewCollector(cfg.Interface)
	spikeDetector := spikes.NewDetector(cfg)
	alertEngine := alerts.NewEngine(cfg)
	logger, err := logging.NewLogger(cfg.LogDir)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Close()

	rotator := storage.NewRotator(cfg.LogDir, cfg.RetentionDays)
	rotator.Start()
	defer rotator.Stop()

	scriptRunner := scripts.NewRunner(cfg)

	var lastSnapshot metrics.MetricsSnapshot
	var lastWriteTime time.Time

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(cfg.SampleInterval())
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			return
		case <-ticker.C:
			snap, err := collector.Collect()
			if err != nil {
				log.Printf("metrics collect error: %v", err)
				continue
			}

			spikeTypes := spikeDetector.Detect(snap, lastSnapshot)
			if len(spikeTypes) > 0 {
				if err := logger.LogSpike(snap, spikeTypes); err != nil {
					log.Printf("log spike error: %v", err)
				}
			}

			alertTypes := alertEngine.Detect(snap, lastSnapshot)
			if len(alertTypes) > 0 {
				if err := logger.LogAlert(snap, alertTypes); err != nil {
					log.Printf("log alert error: %v", err)
				}

				if alertEngine.ShouldExecuteScripts(alertTypes) {
					go func(types []string, snapshot metrics.MetricsSnapshot) {
						if err := scriptRunner.Execute(types, snapshot); err != nil {
							log.Printf("script execution error: %v", err)
						}
					}(alertTypes, snap)
				}
			}

			now := time.Now()
			if lastWriteTime.IsZero() || now.Sub(lastWriteTime) >= cfg.CollectionInterval() {
				if err := logger.LogSample(snap); err != nil {
					log.Printf("log sample error: %v", err)
				}
				lastWriteTime = now
			}

			lastSnapshot = snap
		}
	}
}

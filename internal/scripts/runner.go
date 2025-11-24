package scripts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"system-sentinel/internal/config"
	"system-sentinel/internal/metrics"
)

type Runner struct {
	cfg *config.Config
}

func NewRunner(cfg *config.Config) *Runner {
	return &Runner{cfg: cfg}
}

func (r *Runner) Execute(alertTypes []string, snap metrics.MetricsSnapshot) error {
	if err := r.writeEnvFile(alertTypes, snap); err != nil {
		return fmt.Errorf("failed to write env file: %w", err)
	}

	scripts, err := r.findScripts()
	if err != nil {
		return fmt.Errorf("failed to find scripts: %w", err)
	}

	env := r.buildEnv(alertTypes, snap)

	for _, script := range scripts {
		if err := r.executeScript(script, env); err != nil {
			return fmt.Errorf("script %s failed: %w", script, err)
		}
	}

	return nil
}

func (r *Runner) writeEnvFile(alertTypes []string, snap metrics.MetricsSnapshot) error {
	env := r.buildEnv(alertTypes, snap)

	file, err := os.Create(r.cfg.Scripts.EnvFile)
	if err != nil {
		return err
	}
	defer file.Close()

	for key, value := range env {
		if _, err := fmt.Fprintf(file, "%s=%s\n", key, value); err != nil {
			return err
		}
	}

	return nil
}

func (r *Runner) buildEnv(alertTypes []string, snap metrics.MetricsSnapshot) map[string]string {
	metric := "multi"
	if len(alertTypes) == 1 {
		metric = alertTypes[0]
	}

	env := map[string]string{
		"SYS_TIMESTAMP":        snap.Timestamp.Format(time.RFC3339),
		"SYS_EVENT_TYPE":       "alert",
		"SYS_EVENT_METRIC":     metric,
		"SYS_CPU_USAGE":        strconv.FormatFloat(snap.CPUUsagePercent, 'f', 2, 64),
		"SYS_MEM_USED_PERCENT": strconv.FormatFloat(snap.MemUsedPercent, 'f', 2, 64),
		"SYS_MEM_USED_BYTES":   strconv.FormatUint(snap.MemUsedBytes, 10),
		"SYS_MEM_TOTAL_BYTES":  strconv.FormatUint(snap.MemTotalBytes, 10),
		"SYS_NET_INTERFACE":    snap.NetInterface,
		"SYS_NET_RX_BPS":       strconv.FormatFloat(snap.NetRxBytesPS, 'f', 2, 64),
		"SYS_NET_TX_BPS":       strconv.FormatFloat(snap.NetTxBytesPS, 'f', 2, 64),
		"SYS_NET_RX_MBPS":      strconv.FormatFloat(snap.NetRxMbps, 'f', 2, 64),
		"SYS_NET_TX_MBPS":      strconv.FormatFloat(snap.NetTxMbps, 'f', 2, 64),
	}

	return env
}

func (r *Runner) findScripts() ([]string, error) {
	entries, err := os.ReadDir(r.cfg.Scripts.Dir)
	if err != nil {
		return nil, err
	}

	var scripts []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".sh") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.Mode()&0111 == 0 {
			continue
		}

		scripts = append(scripts, filepath.Join(r.cfg.Scripts.Dir, entry.Name()))
	}

	return scripts, nil
}

func (r *Runner) executeScript(scriptPath string, env map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.cfg.Scripts.TimeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/bash", scriptPath)
	cmd.Env = os.Environ()

	for key, value := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("exit code %d", exitErr.ExitCode())
		}
		return err
	}

	return nil
}

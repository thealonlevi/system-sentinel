package alerts

import (
	"sync"
	"time"

	"system-sentinel/internal/config"
	"system-sentinel/internal/metrics"
)

type Engine struct {
	cfg       *config.Config
	lastFired map[string]time.Time
	mu        sync.RWMutex
}

func NewEngine(cfg *config.Config) *Engine {
	return &Engine{
		cfg:       cfg,
		lastFired: make(map[string]time.Time),
	}
}

func (e *Engine) Detect(current, previous metrics.MetricsSnapshot) []string {
	var alerts []string

	if e.cfg.Alerts.CPU.Enabled {
		if e.detectCPUAlert(current, previous) {
			alerts = append(alerts, "cpu")
		}
	}

	if e.cfg.Alerts.Memory.Enabled {
		if e.detectMemoryAlert(current) {
			alerts = append(alerts, "memory")
		}
	}

	if e.cfg.Alerts.Network.Enabled {
		if e.detectNetworkAlert(current) {
			alerts = append(alerts, "network")
		}
	}

	return alerts
}

func (e *Engine) ShouldExecuteScripts(alertTypes []string) bool {
	if len(alertTypes) == 0 {
		return false
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	debounceDur := time.Duration(e.cfg.Scripts.DebounceSec) * time.Second

	shouldExecute := false
	for _, alertType := range alertTypes {
		lastTime, exists := e.lastFired[alertType]
		if !exists || now.Sub(lastTime) >= debounceDur {
			shouldExecute = true
			e.lastFired[alertType] = now
		}
	}

	return shouldExecute
}

func (e *Engine) detectCPUAlert(current, previous metrics.MetricsSnapshot) bool {
	cfg := e.cfg.Alerts.CPU

	if current.CPUUsagePercent >= cfg.AbsoluteThreshold {
		return true
	}

	if previous.CPUUsagePercent > 0 && cfg.RelativeThreshold > 0 {
		relativeChange := ((current.CPUUsagePercent - previous.CPUUsagePercent) / previous.CPUUsagePercent) * 100.0
		if relativeChange >= cfg.RelativeThreshold {
			return true
		}
	}

	return false
}

func (e *Engine) detectMemoryAlert(current metrics.MetricsSnapshot) bool {
	cfg := e.cfg.Alerts.Memory
	return current.MemUsedPercent >= cfg.AbsoluteThreshold
}

func (e *Engine) detectNetworkAlert(current metrics.MetricsSnapshot) bool {
	cfg := e.cfg.Alerts.Network
	return current.NetRxMbps >= cfg.RxMbpsThreshold || current.NetTxMbps >= cfg.TxMbpsThreshold
}

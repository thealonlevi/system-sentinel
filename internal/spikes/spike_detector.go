package spikes

import (
	"system-sentinel/internal/config"
	"system-sentinel/internal/metrics"
)

type Detector struct {
	cfg *config.Config
}

func NewDetector(cfg *config.Config) *Detector {
	return &Detector{cfg: cfg}
}

func (d *Detector) Detect(current, previous metrics.MetricsSnapshot) []string {
	var spikes []string

	if d.cfg.Spikes.CPU.Enabled {
		if d.detectCPUSpike(current, previous) {
			spikes = append(spikes, "cpu")
		}
	}

	if d.cfg.Spikes.Memory.Enabled {
		if d.detectMemorySpike(current, previous) {
			spikes = append(spikes, "memory")
		}
	}

	if d.cfg.Spikes.Network.Enabled {
		if d.detectNetworkSpike(current, previous) {
			spikes = append(spikes, "network")
		}
	}

	return spikes
}

func (d *Detector) detectCPUSpike(current, previous metrics.MetricsSnapshot) bool {
	cfg := d.cfg.Spikes.CPU

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

func (d *Detector) detectMemorySpike(current, previous metrics.MetricsSnapshot) bool {
	cfg := d.cfg.Spikes.Memory

	if current.MemUsedPercent >= cfg.AbsoluteThreshold {
		return true
	}

	if previous.MemUsedPercent > 0 && cfg.RelativeThreshold > 0 {
		relativeChange := ((current.MemUsedPercent - previous.MemUsedPercent) / previous.MemUsedPercent) * 100.0
		if relativeChange >= cfg.RelativeThreshold {
			return true
		}
	}

	return false
}

func (d *Detector) detectNetworkSpike(current, previous metrics.MetricsSnapshot) bool {
	cfg := d.cfg.Spikes.Network

	if current.NetRxMbps >= cfg.RxMbpsThreshold {
		return true
	}

	if current.NetTxMbps >= cfg.TxMbpsThreshold {
		return true
	}

	if previous.NetRxMbps > 0 && cfg.RelativeThreshold > 0 {
		rxChange := ((current.NetRxMbps - previous.NetRxMbps) / previous.NetRxMbps) * 100.0
		if rxChange >= cfg.RelativeThreshold {
			return true
		}
	}

	if previous.NetTxMbps > 0 && cfg.RelativeThreshold > 0 {
		txChange := ((current.NetTxMbps - previous.NetTxMbps) / previous.NetTxMbps) * 100.0
		if txChange >= cfg.RelativeThreshold {
			return true
		}
	}

	return false
}

package metrics

import "time"

type MetricsSnapshot struct {
	Timestamp       time.Time
	CPUUsagePercent float64
	MemUsedPercent  float64
	MemUsedBytes    uint64
	MemTotalBytes   uint64
	NetInterface    string
	NetRxBytesPS    float64
	NetTxBytesPS    float64
	NetRxMbps       float64
	NetTxMbps       float64
}

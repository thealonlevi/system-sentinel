package metrics

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Collector struct {
	interfaceName  string
	prevCPUStats   cpuStats
	prevNetStats   netStats
	prevSampleTime time.Time
	initialized    bool
}

type cpuStats struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
}

type netStats struct {
	rxBytes uint64
	txBytes uint64
}

func NewCollector(interfaceName string) *Collector {
	return &Collector{
		interfaceName: interfaceName,
	}
}

func (c *Collector) Collect() (MetricsSnapshot, error) {
	now := time.Now()
	snap := MetricsSnapshot{
		Timestamp:    now,
		NetInterface: c.interfaceName,
	}

	cpuUsage, err := c.collectCPU()
	if err != nil {
		return snap, fmt.Errorf("cpu: %w", err)
	}
	snap.CPUUsagePercent = cpuUsage

	memInfo, err := c.collectMemory()
	if err != nil {
		return snap, fmt.Errorf("memory: %w", err)
	}
	snap.MemTotalBytes = memInfo.total
	snap.MemUsedBytes = memInfo.total - memInfo.available
	snap.MemUsedPercent = float64(snap.MemUsedBytes) / float64(memInfo.total) * 100.0

	netInfo, err := c.collectNetwork(now)
	if err != nil {
		return snap, fmt.Errorf("network: %w", err)
	}
	snap.NetRxBytesPS = netInfo.rxBytesPS
	snap.NetTxBytesPS = netInfo.txBytesPS
	snap.NetRxMbps = netInfo.rxBytesPS * 8.0 / 1_000_000.0
	snap.NetTxMbps = netInfo.txBytesPS * 8.0 / 1_000_000.0

	return snap, nil
}

func (c *Collector) collectCPU() (float64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, fmt.Errorf("empty /proc/stat")
	}

	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 8 || fields[0] != "cpu" {
		return 0, fmt.Errorf("invalid cpu line")
	}

	var stats cpuStats
	stats.user, _ = strconv.ParseUint(fields[1], 10, 64)
	stats.nice, _ = strconv.ParseUint(fields[2], 10, 64)
	stats.system, _ = strconv.ParseUint(fields[3], 10, 64)
	stats.idle, _ = strconv.ParseUint(fields[4], 10, 64)
	stats.iowait, _ = strconv.ParseUint(fields[5], 10, 64)
	stats.irq, _ = strconv.ParseUint(fields[6], 10, 64)
	stats.softirq, _ = strconv.ParseUint(fields[7], 10, 64)
	if len(fields) > 8 {
		stats.steal, _ = strconv.ParseUint(fields[8], 10, 64)
	}

	if !c.initialized {
		c.prevCPUStats = stats
		c.initialized = true
		return 0.0, nil
	}

	totalPrev := c.prevCPUStats.user + c.prevCPUStats.nice + c.prevCPUStats.system +
		c.prevCPUStats.idle + c.prevCPUStats.iowait + c.prevCPUStats.irq +
		c.prevCPUStats.softirq + c.prevCPUStats.steal

	totalCurr := stats.user + stats.nice + stats.system + stats.idle +
		stats.iowait + stats.irq + stats.softirq + stats.steal

	idlePrev := c.prevCPUStats.idle + c.prevCPUStats.iowait
	idleCurr := stats.idle + stats.iowait

	totalDelta := totalCurr - totalPrev
	idleDelta := idleCurr - idlePrev

	if totalDelta == 0 {
		c.prevCPUStats = stats
		return 0.0, nil
	}

	usage := (1.0 - float64(idleDelta)/float64(totalDelta)) * 100.0
	c.prevCPUStats = stats

	return usage, nil
}

type memoryInfo struct {
	total     uint64
	available uint64
}

func (c *Collector) collectMemory() (memoryInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return memoryInfo{}, err
	}
	defer file.Close()

	var info memoryInfo
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				info.total = val * 1024
			}
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, _ := strconv.ParseUint(fields[1], 10, 64)
				info.available = val * 1024
			}
		}
		if info.total > 0 && info.available > 0 {
			break
		}
	}

	if info.total == 0 {
		return memoryInfo{}, fmt.Errorf("could not read MemTotal")
	}
	if info.available == 0 {
		info.available = info.total
	}

	return info, nil
}

type networkInfo struct {
	rxBytesPS float64
	txBytesPS float64
}

func (c *Collector) collectNetwork(now time.Time) (networkInfo, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return networkInfo{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var rxBytes, txBytes uint64
	found := false

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, c.interfaceName) {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		interfacePart := fields[0]
		if !strings.HasPrefix(interfacePart, c.interfaceName+":") {
			continue
		}

		rxBytes, _ = strconv.ParseUint(fields[1], 10, 64)
		txBytes, _ = strconv.ParseUint(fields[9], 10, 64)
		found = true
		break
	}

	if !found {
		return networkInfo{}, fmt.Errorf("interface %s not found", c.interfaceName)
	}

	if !c.initialized || c.prevSampleTime.IsZero() {
		c.prevNetStats = netStats{rxBytes: rxBytes, txBytes: txBytes}
		c.prevSampleTime = now
		return networkInfo{rxBytesPS: 0, txBytesPS: 0}, nil
	}

	deltaTime := now.Sub(c.prevSampleTime).Seconds()
	if deltaTime <= 0 {
		deltaTime = 1.0
	}

	rxDelta := rxBytes - c.prevNetStats.rxBytes
	txDelta := txBytes - c.prevNetStats.txBytes

	rxBytesPS := float64(rxDelta) / deltaTime
	txBytesPS := float64(txDelta) / deltaTime

	c.prevNetStats = netStats{rxBytes: rxBytes, txBytes: txBytes}
	c.prevSampleTime = now

	return networkInfo{rxBytesPS: rxBytesPS, txBytesPS: txBytesPS}, nil
}

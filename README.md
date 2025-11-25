# System Sentinel

Small Linux watchdog for machines that need continuous metrics, spike detection, and programmable alert hooks.

`system-sentinel` samples core host metrics every second, keeps structured NDJSON logs, prunes them automatically, and triggers shell scripts when alerts fire so you can fan out to webhooks, paging pipelines, or chat bots.

Language: Go · Status: v1.0.0 · License: MIT

## Features

- **Kernel-backed metrics** – Reads `/proc/stat`, `/proc/meminfo`, and `/proc/net/dev` to track CPU utilization, RAM usage (bytes and percent), and per-interface network throughput (bytes/s and Mbps).
- **Configurable spike + alert engines** – Separate CPU, memory, and network thresholds for spikes (log-only) and alerts (log + script) with both absolute and relative rules.
- **Script runner with rich env** – Executes every executable `.sh` in the configured directory, injects `SYS_*` metrics plus any custom key/value pairs from the config `env:` map, writes the same set to a `.env` file, and enforces per-script timeouts and debounce windows.
- **Daily NDJSON logs** – Streams `sample`, `spike`, and `alert` entries to `metrics-YYYY-MM-DD.ndjson` with the full snapshot embedded, making it easy to grep or feed into `jq`.
- **Automated retention** – Background rotator purges log files older than `retention_days`.
- **Systemd-friendly** – Ships with install/uninstall scripts and a unit file that builds, installs, and manages the service under `/usr/local/bin/system-sentinel`.

## Architecture Overview

At startup the daemon loads `config.yaml`, initializes logging and retention, instantiates the metrics collector, spike and alert engines, and the script runner. A ticker drives the main loop: collect metrics → detect spikes → write NDJSON (spikes and alerts include reasons) → when alerts fire, optionally execute scripts → periodically write full samples and rotate logs.

Key packages:

- `internal/config`: YAML parsing, defaults, validation, and typed accessors.
- `internal/metrics`: Collector that produces `MetricsSnapshot` structs (timestamp, CPU%, RAM bytes/% , interface throughput).
- `internal/spikes` and `internal/alerts`: Threshold engines for spike and alert detection with absolute/relative logic and debounce handling.
- `internal/scripts`: Script discovery, env construction, `.env` writer, and `/bin/bash` execution with per-script timeouts.
- `internal/logging`: NDJSON writer with daily rotation.
- `internal/storage`: Retention rotator that deletes expired log files.

## Installation

### Requirements

- Go 1.21+
- Linux with `/proc` metrics, `/bin/bash`, and systemd (unit file targets `multi-user.target`)

### Install via script

```bash
git clone https://github.com/thealonlevi/system-sentinel.git
cd system-sentinel
bash packaging/install.sh
```

The installer will:

1. Build `./cmd/system-sentinel` and copy it to `/usr/local/bin/system-sentinel`.
2. Create `/etc/system-sentinel` (preserving an existing `config.yaml` if present).
3. Copy every `.sh` from `sh/` into `/etc/system-sentinel/sh/` (skipping files that already exist) and mark them executable.
4. Create `/var/log/system-sentinel`.
5. Install `packaging/systemd/system-sentinel.service`, reload systemd, and enable/start the service.

### Manual build and deploy

```bash
git clone https://github.com/thealonlevi/system-sentinel.git
cd system-sentinel
go build -o bin/system-sentinel ./cmd/system-sentinel
```

Then copy `bin/system-sentinel` anywhere on your `$PATH`, place a config at `/etc/system-sentinel/config.yaml`, and optionally install the provided systemd unit plus scripts.

## Configuration

`system-sentinel` looks for `/etc/system-sentinel/config.yaml` by default (override with `-config /path/to/file`). Example:

```yaml
sample_interval_sec: 1
collection_interval_sec: 60
log_dir: /var/log/system-sentinel
retention_days: 30
interface: eno1

spikes:
  cpu:
    enabled: true
    absolute_threshold: 80.0
    relative_threshold: 50.0
  memory:
    enabled: true
    absolute_threshold: 85.0
    relative_threshold: 20.0
  network:
    enabled: true
    rx_mbps_threshold: 100.0
    tx_mbps_threshold: 100.0
    relative_threshold: 100.0

alerts:
  cpu:
    enabled: true
    absolute_threshold: 75.0
    relative_threshold: 0.0
  memory:
    enabled: true
    absolute_threshold: 75.0
  network:
    enabled: true
    rx_mbps_threshold: 1000.0
    tx_mbps_threshold: 1000.0

env:
  SYS_PUBLIC_IP: "127.0.0.1"
  FLASH_ALERT_WEBHOOK_URL: ""
  EVENTS_HMAC_SECRET: ""
  SERVICE: "api"

scripts:
  dir: /etc/system-sentinel/sh
  env_file: /etc/system-sentinel/.env
  debounce_sec: 60
  timeout_sec: 30
  enabled: true
```

Field reference:

- `sample_interval_sec` – Frequency of metric collection (seconds). Default 1.
- `collection_interval_sec` – How often to write a `sample` log entry. Defaults to 60 seconds.
- `log_dir` / `retention_days` – NDJSON location and retention horizon for the rotator.
- `interface` – Network device name passed to `/proc/net/dev`.
- `spikes.*` – Per-metric spike detection: absolute percentage thresholds and optional relative change windows. Spikes are logged only.
- `alerts.*` – Alert thresholds; hitting them logs an alert and can execute scripts.
- `env` – Arbitrary key/value pairs exported to scripts. Config values override built-in `SYS_*` keys if they collide.
- `scripts.dir` – Directory scanned for executable `.sh` files. Only suffix `.sh` files with the execute bit run.
- `scripts.env_file` – Path to the generated `.env` file mirroring the runtime env map.
- `scripts.debounce_sec` – Minimum time between runs per alert type inside `alerts.ShouldExecuteScripts`.
- `scripts.timeout_sec` – Per-script execution timeout enforced via `context.WithTimeout`.
- `scripts.enabled` – Master toggle for script execution.

There are no secret environment variable overrides; all behavior is driven by YAML.

## Running as a Service

The unit installs as `system-sentinel.service` (Type=simple). Common commands:

```bash
sudo systemctl status system-sentinel
sudo systemctl start system-sentinel
sudo systemctl stop system-sentinel
sudo systemctl restart system-sentinel
sudo systemctl enable system-sentinel
```

The unit runs under the default systemd service user (typically `root`); adjust the file if you need a dedicated account.

## Logs & Data

- **Location:** `log_dir` (default `/var/log/system-sentinel`).
- **Naming:** `metrics-YYYY-MM-DD.ndjson` (UTC date). Logger rotates automatically at midnight UTC.
- **Format:** Each line is a JSON object containing `timestamp`, `type` (`sample`, `spike`, `alert`), `metric` (cpu/memory/network/multi), optional `reasons` array, and an embedded `metrics` snapshot with CPU%, memory bytes/percent, interface name, RX/TX bytes per second, and RX/TX Mbps.
- **Retention:** `internal/storage.Rotator` scans every six hours and deletes files older than `retention_days`.

Tail logs live:

```bash
sudo tail -f /var/log/system-sentinel/metrics-$(date -u +%Y-%m-%d).ndjson
```

Filter alerts with `jq`:

```bash
sudo jq 'select(.type=="alert")' /var/log/system-sentinel/metrics-*.ndjson
```

## Spike Detection & Scripts

### Detection

- **CPU spikes/alerts:** Trigger when instantaneous usage meets `absolute_threshold` or the relative increase from the previous sample exceeds `relative_threshold`.
- **Memory spikes/alerts:** Same logic but based on `MemUsedPercent`.
- **Network spikes/alerts:** Compare RX/TX Mbps against absolute thresholds and optional relative change percentages.
- Spike hits are logged only; alert hits log **and** can trigger scripts when `scripts.enabled` is true and `alerts.ShouldExecuteScripts` allows it (per-metric debounce window).

### Script execution

- `internal/scripts.Runner` scans `scripts.dir` for executable `.sh` files and runs them sequentially via `/bin/bash`. Execution stops on the first failure; the error is logged.
- Each run receives the base environment plus static entries from `env:`; the same map is written as `KEY=value` lines to `scripts.env_file`.
- Built-in keys:
  - `SYS_TIMESTAMP`
  - `SYS_EVENT_TYPE` (always `"alert"`)
  - `SYS_EVENT_METRIC` (`cpu`, `memory`, `network`, `multi`)
  - `SYS_CPU_USAGE`
  - `SYS_MEM_USED_PERCENT`
  - `SYS_MEM_USED_BYTES`
  - `SYS_MEM_TOTAL_BYTES`
  - `SYS_NET_INTERFACE`
  - `SYS_NET_RX_BPS`
  - `SYS_NET_TX_BPS`
  - `SYS_NET_RX_MBPS`
  - `SYS_NET_TX_MBPS`
- Any key defined under `env:` (e.g., `SYS_PUBLIC_IP`, webhook URLs, HMAC secrets, service tags) is added and can override defaults.
- Scripts should be owned by a trusted user, have mode `0755`, and avoid long-running tasks because of the enforced timeout.

## Security Considerations

- The default systemd unit runs as root; restrict execution rights or modify the unit if you prefer a limited user.
- `/etc/system-sentinel/config.yaml`, the generated `.env`, and scripts may contain secrets (webhook URLs, HMAC keys). Set permissions appropriately (e.g., `chmod 600` for configs that include secrets).
- Only place trusted scripts in `/etc/system-sentinel/sh`; each alert executes arbitrary code as the service user.
- Outbound hooks should validate TLS certificates or perform their own signing (see `sh/system_sentinel_alert.sh` for an HMAC example).

## Uninstall

```bash
cd system-sentinel
bash packaging/uninstall.sh
```

The script stops and disables the service, removes `/usr/local/bin/system-sentinel`, deletes the systemd unit, wipes `/etc/system-sentinel`, and removes `/var/log/system-sentinel`.

## Troubleshooting

- **Service will not start:** `sudo systemctl status system-sentinel` then `sudo journalctl -u system-sentinel -n 100` to inspect Go log output.
- **Permission errors writing logs:** Ensure `log_dir` exists and is writable by the service user (default root). The installer creates `/var/log/system-sentinel` with liberal permissions.
- **No spikes or alerts ever trigger:** Lower absolute/relative thresholds in `config.yaml`, confirm the `interface` name matches `ip link show`.
- **Too many spikes:** Increase thresholds or lengthen `debounce_sec` so scripts are not spammed.
- **Scripts never run:** Verify `scripts.enabled: true`, scripts are executable, and look for errors in `journalctl` indicating timeouts or exit codes.
- **Network stats zero:** Interface names in `/proc/net/dev` may differ from predictable names (`eth0` vs `eno1`). Update the `interface` field and restart the service.

## Contributing & Development

- Run unit checks: `go test ./...`.
- Build locally without systemd: `go build -o bin/system-sentinel ./cmd/system-sentinel` then `./bin/system-sentinel -config ./config.yaml`.
- Pull requests should include updated docs when config or runtime behavior changes.

## Versioning & License

- Current release: **v1.0.0**
- License: [MIT](./LICENSE)

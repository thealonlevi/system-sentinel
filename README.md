# system-sentinel

A production-ready Linux daemon for continuous system metrics monitoring with spike detection, alerting, and script execution hooks.

## Features

* Continuous CPU, memory, and network monitoring
* NDJSON log output with daily rotation
* Separate spike detection (log-only warnings) and alert thresholds
* Shell script hooks with environment variable injection
* Automatic log retention management
* Systemd integration

## Installation

1. Clone this repository:
   ```bash
   git clone <repository-url>
   cd system-sentinel
   ```

2. Run the install script:
   ```bash
   bash packaging/install.sh
   ```

The install script will:
* Build the binary
* Install it to `/usr/local/bin/system-sentinel`
* Create configuration directory at `/etc/system-sentinel`
* Set up systemd service
* Enable and start the daemon

## Configuration

Edit `/etc/system-sentinel/config.yaml` to customize behavior:

### Global Settings

* `sample_interval_sec`: How often to collect metrics (default: 1)
* `collection_interval_sec`: How often to write sample logs (default: 60)
* `log_dir`: Directory for log files (default: `/var/log/system-sentinel`)
* `retention_days`: Keep logs for this many days (default: 30)
* `interface`: Network interface to monitor (default: `eth0`)

### Spikes

Spikes are logged but do not trigger scripts. Configure thresholds for:
* `cpu`: Absolute percentage or relative change
* `memory`: Absolute percentage or relative change
* `network`: Rx/Tx Mbps thresholds or relative change

### Alerts

Alerts are logged and can trigger scripts. Configure thresholds for:
* `cpu`: Absolute percentage or relative change
* `memory`: Absolute percentage
* `network`: Rx/Tx Mbps thresholds

### Scripts

* `dir`: Directory containing executable `.sh` scripts (default: `/etc/system-sentinel/sh`)
* `env_file`: Path to write environment variables (default: `/etc/system-sentinel/.env`)
* `debounce_sec`: Minimum seconds between script executions per alert type (default: 60)
* `timeout_sec`: Maximum execution time per script (default: 30)

## Custom Alert Scripts

Create executable `.sh` files in `/etc/system-sentinel/sh/`. Each script receives these environment variables:

* `SYS_TIMESTAMP`: Event timestamp (RFC3339)
* `SYS_EVENT_TYPE`: Always "alert"
* `SYS_EVENT_METRIC`: "cpu", "memory", "network", or "multi"
* `SYS_CPU_USAGE`: CPU usage percentage
* `SYS_MEM_USED_PERCENT`: Memory used percentage
* `SYS_MEM_USED_BYTES`: Memory used in bytes
* `SYS_MEM_TOTAL_BYTES`: Total memory in bytes
* `SYS_NET_INTERFACE`: Network interface name
* `SYS_NET_RX_BPS`: Receive bytes per second
* `SYS_NET_TX_BPS`: Transmit bytes per second
* `SYS_NET_RX_MBPS`: Receive Mbps
* `SYS_NET_TX_MBPS`: Transmit Mbps

### Example: Webhook Alert

```bash
#!/usr/bin/env bash
curl -X POST https://your-webhook-url.com/alerts \
  -H "Content-Type: application/json" \
  -d "{
    \"timestamp\": \"$SYS_TIMESTAMP\",
    \"metric\": \"$SYS_EVENT_METRIC\",
    \"cpu\": $SYS_CPU_USAGE,
    \"memory\": $SYS_MEM_USED_PERCENT
  }"
```

Save as `/etc/system-sentinel/sh/webhook_alert.sh` and make it executable:
```bash
chmod +x /etc/system-sentinel/sh/webhook_alert.sh
```

## Usage

### View Logs

Logs are written as NDJSON (newline-delimited JSON) to daily files:
```
/var/log/system-sentinel/metrics-YYYY-MM-DD.ndjson
```

View recent logs:
```bash
tail -f /var/log/system-sentinel/metrics-$(date +%Y-%m-%d).ndjson
```

Filter with `jq`:
```bash
cat /var/log/system-sentinel/metrics-*.ndjson | jq 'select(.type=="alert")'
```

### Service Management

```bash
systemctl status system-sentinel
systemctl restart system-sentinel
systemctl stop system-sentinel
```

### Uninstallation

```bash
bash packaging/uninstall.sh
```

## Log Format

Each log entry is a JSON object with:

* `timestamp`: ISO8601 timestamp
* `type`: "sample", "spike", or "alert"
* `metric`: "cpu", "memory", "network", or "multi"
* `reasons`: Array of metric types that triggered (for spikes/alerts)
* `metrics`: Full metrics snapshot object

## License

MIT


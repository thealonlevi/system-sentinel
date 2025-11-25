#!/usr/bin/env bash
set -euo pipefail

URL="${FLASH_ALERT_WEBHOOK_URL:-}"
SECRET="${EVENTS_HMAC_SECRET:-}"
SERVICE="${SERVICE:-api}"

if [ -z "$URL" ] || [ -z "$SECRET" ]; then
  exit 0
fi

ts_ms="$(date +%s%3N)"
idempotency_key="$(uuidgen 2>/dev/null || echo "${ts_ms}-$$")"
host="$(hostname)"

server_ip="${SYS_PUBLIC_IP:-$host}"

metric="${SYS_EVENT_METRIC:-unknown}"
sys_timestamp="${SYS_TIMESTAMP:-}"
sys_event_type="${SYS_EVENT_TYPE:-}"

cpu="${SYS_CPU_USAGE:-unknown}"
mem_percent="${SYS_MEM_USED_PERCENT:-unknown}"
mem_used="${SYS_MEM_USED_BYTES:-unknown}"
mem_total="${SYS_MEM_TOTAL_BYTES:-unknown}"
net_if="${SYS_NET_INTERFACE:-unknown}"
net_rx="${SYS_NET_RX_MBPS:-unknown}"
net_tx="${SYS_NET_TX_MBPS:-unknown}"

ts_iso_now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

body=$(printf '{"severity":"HIGH","topic":"system-sentinel.alert","alert_key":"system-sentinel:%s:%s","message":"System sentinel alert for %s on %s","labels":{"env":"prod","service":"system-sentinel","source_host":"%s","server_ip":"%s","event_metric":"%s"},"details":{"sys_timestamp":"%s","sys_event_type":"%s","cpu_usage":"%s","mem_used_percent":"%s","mem_used_bytes":"%s","mem_total_bytes":"%s","net_interface":"%s","net_rx_mbps":"%s","net_tx_mbps":"%s"},"occurred_at":"%s","idempotency_key":"%s"}' \
  "$metric" "$server_ip" \
  "$metric" "$server_ip" \
  "$host" "$server_ip" "$metric" \
  "$sys_timestamp" "$sys_event_type" "$cpu" "$mem_percent" "$mem_used" "$mem_total" "$net_if" "$net_rx" "$net_tx" \
  "$ts_iso_now" "$idempotency_key")

msg="${ts_ms}.${body}"
sig=$(printf '%s' "$msg" | openssl dgst -binary -sha256 -hmac "$SECRET" | base64)

curl -sS -X POST "$URL" \
  -H "content-type: application/json" \
  -H "x-service: $SERVICE" \
  -H "x-timestamp: $ts_ms" \
  -H "x-signature: $sig" \
  -d "$body" >/dev/null || true


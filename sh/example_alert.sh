#!/usr/bin/env bash
echo "$(date -Iseconds) ALERT: $SYS_EVENT_METRIC - CPU: ${SYS_CPU_USAGE}% MEM: ${SYS_MEM_USED_PERCENT}% NET: ${SYS_NET_RX_MBPS}Mbps/${SYS_NET_TX_MBPS}Mbps" >> /var/log/system-sentinel/alerts.log


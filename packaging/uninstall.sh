#!/bin/bash
set -e

echo "Stopping and disabling service..."
sudo systemctl disable --now system-sentinel 2>/dev/null || true

echo "Removing binary..."
sudo rm -f /usr/local/bin/system-sentinel

echo "Removing systemd unit..."
sudo rm -f /etc/systemd/system/system-sentinel.service
sudo systemctl daemon-reload

echo "Removing config directory..."
sudo rm -rf /etc/system-sentinel

echo "Removing log directory..."
sudo rm -rf /var/log/system-sentinel

echo "Uninstallation complete!"


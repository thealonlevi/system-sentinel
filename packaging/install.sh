#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$REPO_ROOT"

echo "Building system-sentinel..."
go build -o bin/system-sentinel ./cmd/system-sentinel

echo "Installing binary..."
sudo mkdir -p /usr/local/bin
sudo cp bin/system-sentinel /usr/local/bin/system-sentinel
sudo chmod +x /usr/local/bin/system-sentinel

echo "Creating config directory..."
sudo mkdir -p /etc/system-sentinel

if [ ! -f /etc/system-sentinel/config.yaml ]; then
    echo "Copying default config..."
    sudo cp config.yaml /etc/system-sentinel/config.yaml
fi

echo "Creating scripts directory..."
sudo mkdir -p /etc/system-sentinel/sh

echo "Installing scripts from sh/..."
for script in sh/*.sh; do
    [ -f "$script" ] || continue
    script_name="$(basename "$script")"
    target="/etc/system-sentinel/sh/$script_name"
    if [ ! -f "$target" ]; then
        echo "Installing script $script_name..."
        sudo cp "$script" "$target"
        sudo chmod +x "$target"
    else
        echo "Script $script_name already exists, skipping."
    fi
done

echo "Creating log directory..."
sudo mkdir -p /var/log/system-sentinel

echo "Installing systemd unit..."
sudo cp packaging/systemd/system-sentinel.service /etc/systemd/system/system-sentinel.service

echo "Reloading systemd..."
sudo systemctl daemon-reload

echo "Enabling and starting service..."
sudo systemctl enable --now system-sentinel

echo "Installation complete!"
echo "Check status with: systemctl status system-sentinel"
echo "View logs with: tail -f /var/log/system-sentinel/metrics-*.ndjson"

#!/bin/bash

# Install dependencies and build
sudo apt update
sudo apt install golang-go -y
go mod init binfinite-rpi
go get golang.org/x/sys/unix@latest
go mod tidy
go build -o binfinite-rpi binfinite-rpi.go

# Deploy binary
sudo mv binfinite-rpi /usr/local/bin/
sudo chmod +x /usr/local/bin/binfinite-rpi
sudo setcap cap_net_raw+ep /usr/local/bin/binfinite-rpi

# Create systemd service
sudo tee /etc/systemd/system/binfinite-rpi.service > /dev/null <<EOL
[Unit]
Description=Binfinite RPI for Halo Infinite Server Announcements
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/binfinite-rpi \\
  --iface=wlan0 \\
  --log
Restart=always
RestartSec=5
User=root
AmbientCapabilities=CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_RAW

[Install]
WantedBy=multi-user.target
EOL

# Reload and start service
sudo systemctl daemon-reload
sudo systemctl enable binfinite-rpi.service
sudo systemctl start binfinite-rpi.service
sudo systemctl status binfinite-rpi.service

echo "Setup complete! Edit servers.json and restart service with 'sudo systemctl restart binfinite-rpi.service'"

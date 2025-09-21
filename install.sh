#!/bin/bash

# Check and upgrade Go to 1.23+
GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//g' | cut -d. -f1)
if [ "$GO_VERSION" -lt 1.21 ]; then
  echo "Upgrading Go to 1.23..."
  sudo apt remove golang-go -y
  wget https://go.dev/dl/go1.23.0.linux-arm64.tar.gz
  sudo tar -C /usr/local -xzf go1.23.0.linux-arm64.tar.gz
  rm go1.23.0.linux-arm64.tar.gz
  echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
  source ~/.bashrc
  go version
fi

# Install dependencies and build
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
  --iface=eth0 \\
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

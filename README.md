**Build (Mac → Pi):**

go mod init binfinite-rpi

go get golang.org/x/sys/unix@latest

go mod tidy

CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o binfinitetunnel binfinitetunnel.go

**Deploy on the Pi:**

ssh   user@localhostname

sudo mv ./binfinite-rpi /usr/local/bin/

sudo setcap cap_net_raw+ep /usr/local/bin/binfinite-rpi

**Run on the Pi:**

sudo /usr/local/bin/binfinite-rpi \
  --iface=eth0 \
  --static='[{"name":"SYD SUPERMAX - LINUX","address":"103.214.222.1"},{"name":"MEL SUPERMAX - LINUX","address":"67.219.103.132"}]' \
  --be-prefix=false \
  --log


**To deploy permanently:**

sudo nano /etc/systemd/system/binfinite-rpi.service

[Unit]
Description=Binfinite Tunnel for Halo Infinite Server Announcements
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/binfinite-rpi \
  --iface=eth0 \
  --static='[{"name":"SYD SUPERMAX - LINUX","address":"103.214.222.1"},{"name":"MEL SUPERMAX - LINUX","address":"67.219.103.132"}]' \
  --log
Restart=always
RestartSec=5
User=root
AmbientCapabilities=CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_RAW

[Install]
WantedBy=multi-user.target


**Reload**
sudo systemctl daemon-reload
sudo systemctl restart binfinite-rpi.service
sudo systemctl enable binfinite-rpi.service

**Logs**

journalctl -u binfinitetunnel.service -f

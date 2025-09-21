Build (Mac → Pi/arm64):

go mod init binfinite-rpi
go get golang.org/x/sys/unix@latest
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o binfinitetunnel binfinitetunnel.go

Deploy on the Pi:

ssh   user@localhostname
sudo mv ./binfinite-rpi /usr/local/bin/
sudo setcap cap_net_raw+ep /usr/local/bin/binfinite-rpi

Run on the Pi:

sudo /usr/local/bin/binfinitetunnel \
  --iface=eth0 \
  --static='[{"name":"SYD SUPERMAX - LINUX","address":"103.214.222.1"},{"name":"MEL SUPERMAX - LINUX","address":"67.219.103.132"}]' \
  --be-prefix=false \
  --log



# Binfinitetunnel-RPi

This is a Linux/Raspberry Pi emulation of the original [binfinitetunnel](https://github.com/Jeffx539/binfinitetunnel) script for Halo Infinite server announcements. It forges LAN server announcements to make remote servers appear local to your Xbox/PC on the same network.

## Prerequisites
- Raspberry Pi with Raspberry Pi OS (64-bit recommended).
- Ethernet or Wi-Fi on the same network as your Xbox.
- Basic terminal access (SSH or direct).

## Installation
1. Clone the repo:
   
git clone https://github.com/callmemitch27/binfinitetunnel-rpi.git

cd binfinitetunnel-rpi-

2. Run the install script (installs Go, builds, sets up service):

chmod +x install.sh

./install.sh

- This sets up everything, including a systemd service that runs on boot.

3. Edit the server list (add/remove servers):

nano servers.json

- Example:
  ```json
  [
    {
      "name": "NAME 2",
      "address": "###.###.###.###"
    },
    {
      "name": "NAME 1",
      "address": "###.###.###.###"
    }
  ]

Save and exit (Ctrl+O, Enter, Ctrl+X).

Restart the service to apply changes:

sudo systemctl restart binfinitetunnel.service

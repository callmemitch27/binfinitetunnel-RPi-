package tunnel

import (
	"log"

	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wintun"
)

type Tunnel struct {
	adapter *wintun.Adapter
	session wintun.Session
}

func Init() (*Tunnel, error) {

	var guid windows.GUID
	adapter, err := wintun.CreateAdapter("Binfinite", "Wintun", &guid)
	if err != nil {
		return nil, err
	}

	session, err := adapter.StartSession(0x800000)
	if err != nil {

		return nil, err
	}

	return &Tunnel{adapter: adapter, session: session}, nil

}

func (tun *Tunnel) Broadcast(srcAddr string, payload []byte) {

	pl := BuildUDPPacket(srcAddr, "255.255.255.255", payload)
	packet, err := tun.session.AllocateSendPacket(len(pl))

	if err != nil {
		log.Fatal("Failed to allocate packet", err)
	}
	copy(packet, pl)
	tun.session.SendPacket(packet)
}

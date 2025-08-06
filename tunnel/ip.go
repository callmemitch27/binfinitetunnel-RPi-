package tunnel

// yeah not implementing this
// https://gist.github.com/chrisnc/0ff3d1c20cb6687454b0
import (
	"bytes"
	"encoding/binary"
	"net"

	"golang.org/x/sys/windows"
)

type iphdr struct {
	vhl   uint8
	tos   uint8
	iplen uint16
	id    uint16
	off   uint16
	ttl   uint8
	proto uint8
	csum  uint16
	src   [4]byte
	dst   [4]byte
}

type udphdr struct {
	src  uint16
	dst  uint16
	ulen uint16
	csum uint16
}

// pseudo header used for checksum calculation
type pseudohdr struct {
	ipsrc   [4]byte
	ipdst   [4]byte
	zero    uint8
	ipproto uint8
	plen    uint16
}

func checksum(buf []byte) uint16 {
	sum := uint32(0)

	for ; len(buf) >= 2; buf = buf[2:] {
		sum += uint32(buf[0])<<8 | uint32(buf[1])
	}
	if len(buf) > 0 {
		sum += uint32(buf[0]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	csum := ^uint16(sum)
	/*
	 * From RFC 768:
	 * If the computed checksum is zero, it is transmitted as all ones (the
	 * equivalent in one's complement arithmetic). An all zero transmitted
	 * checksum value means that the transmitter generated no checksum (for
	 * debugging or for higher level protocols that don't care).
	 */
	if csum == 0 {
		csum = 0xffff
	}
	return csum
}

func (h *iphdr) checksum() {
	h.csum = 0
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, h)
	h.csum = checksum(b.Bytes())
}

func (u *udphdr) checksum(ip *iphdr, payload []byte) {
	u.csum = 0
	phdr := pseudohdr{
		ipsrc:   ip.src,
		ipdst:   ip.dst,
		zero:    0,
		ipproto: ip.proto,
		plen:    u.ulen,
	}
	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, &phdr)
	binary.Write(&b, binary.BigEndian, u)
	binary.Write(&b, binary.BigEndian, &payload)
	u.csum = checksum(b.Bytes())
}

func BuildUDPPacket(src string, dst string, payload []byte) []byte {
	ipsrc := net.ParseIP(src)
	ipdst := net.ParseIP(dst)
	ip := iphdr{
		vhl:   0x45,
		tos:   0,
		id:    0x1234,
		off:   0,
		ttl:   255,
		proto: windows.IPPROTO_UDP,
	}
	copy(ip.src[:], ipsrc.To4())
	copy(ip.dst[:], ipdst.To4())

	udp := udphdr{
		src: uint16(7117),
		dst: uint16(7117),
	}

	udplen := 8 + len(payload)
	totalLen := 20 + udplen
	ip.iplen = uint16(totalLen)
	ip.checksum()

	udp.ulen = uint16(udplen)
	udp.checksum(&ip, payload)

	var b bytes.Buffer
	binary.Write(&b, binary.BigEndian, &ip)
	binary.Write(&b, binary.BigEndian, &udp)
	binary.Write(&b, binary.BigEndian, &payload)

	return b.Bytes()

}

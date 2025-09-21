package main

import (
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
	"unicode/utf16"

	"golang.org/x/sys/unix"
)

type Server struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

var (
	masterURL   = flag.String("master", "https://binfinite.lh2.au/servers", "master server list URL")
	staticJSON  = flag.String("static", "", "inline JSON server list (overrides --master)")
	ifaceName   = flag.String("iface", "eth0", "interface to send on (e.g. eth0 or wlan0)")
	logSends    = flag.Bool("log", false, "log each send")
	bePrefix    = flag.Bool("be-prefix", true, "use BigEndian prefix 0x000D (default true, set false for LittleEndian)")
	httpTimeout = 5 * time.Second
)

func mustIPv4(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil || ip.To4() == nil {
		log.Fatalf("invalid IPv4: %s", s)
	}
	return ip.To4()
}

func fetchServers() ([]Server, error) {
	if *staticJSON != "" {
		var s []Server
		if err := json.Unmarshal([]byte(*staticJSON), &s); err != nil {
			return nil, fmt.Errorf("parse --static: %w", err)
		}
		return s, nil
	}
	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(*masterURL)
	if err != nil {
		return nil, fmt.Errorf("master get: %w", err)
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read master: %w", err)
	}
	var out []Server
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("unmarshal master: %w", err)
	}
	return out, nil
}

func stringToUTF16BE(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, 2*len(u))
	for i, cp := range u {
		binary.BigEndian.PutUint16(b[2*i:], cp)
	}
	return b
}

func buildServerPacket(name string) []byte {
	footer := []byte{0x92, 0x87, 0x6d, 0xe1, 0xb1, 0xa5, 0xd4, 0x42, 0xbe, 0x05, 0x09, 0x94, 0x31, 0xb6, 0x37, 0xf0, 0x00, 0x00, 0x40}
	prefix := []byte{0x00, 0x0d}
	if !*bePrefix {
		prefix = []byte{0x0d, 0x00}
	}
	p := prefix
	p = append(p, stringToUTF16BE(name)...)
	p = append(p, 0x00, 0x00)
	p = append(p, footer...)
	return p
}

func encryptPayload(plain []byte) []byte {
	key := []byte{0x2C, 0x0A, 0xA3, 0xD0, 0xAB, 0xB8, 0xEF, 0x33, 0x83, 0x35, 0x04, 0x02, 0xDC, 0x32, 0x8C, 0xA5}
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
		panic(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err)
	}
	enc := gcm.Seal(nil, nonce, plain, nil)
	out := append(nonce, 0xFF, 0x03, 0x43, 0x00)
	out = append(out, enc[len(enc)-16:]...)
	out = append(out, enc[:len(enc)-16]...)
	return out
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
	if csum == 0 {
		csum = 0xffff
	}
	return csum
}

func buildUDPPacket(srcAddr, dstAddr string, payload []byte) []byte {
	ipSrc := mustIPv4(srcAddr)
	ipDst := mustIPv4(dstAddr)
	ip := make([]byte, 20)
	ip[0] = 0x45
	ip[1] = 0
	total := 20 + 8 + len(payload)
	binary.BigEndian.PutUint16(ip[2:4], uint16(total))
	binary.BigEndian.PutUint16(ip[4:6], uint16(rand.Intn(65536)))
	binary.BigEndian.PutUint16(ip[6:8], 0)
	ip[8] = 255 // TTL
	ip[9] = 17  // UDP
	copy(ip[12:16], ipSrc)
	copy(ip[16:20], ipDst)
	ip[10] = 0
	ip[11] = 0
	binary.BigEndian.PutUint16(ip[10:12], checksum(ip))

	udp := make([]byte, 8)
	binary.BigEndian.PutUint16(udp[0:2], 7117)
	binary.BigEndian.PutUint16(udp[2:4], 7117)
	binary.BigEndian.PutUint16(udp[4:6], uint16(8+len(payload)))
	udp[6] = 0
	udp[7] = 0
	tmpUdp := make([]byte, 8)
	copy(tmpUdp, udp)
	binary.BigEndian.PutUint16(udp[6:8], udpChecksum(ip, tmpUdp, payload))

	p := make([]byte, 0, total)
	p = append(p, ip...)
	p = append(p, udp...)
	p = append(p, payload...)
	return p
}

func udpChecksum(ip, udp, payload []byte) uint16 {
	sum := uint32(0)
	// Pseudo header
	sum += uint32(binary.BigEndian.Uint16(ip[12:14]))
	sum += uint32(binary.BigEndian.Uint16(ip[14:16]))
	sum += uint32(binary.BigEndian.Uint16(ip[16:18]))
	sum += uint32(binary.BigEndian.Uint16(ip[18:20]))
	sum += uint32(0x0011)
	sum += uint32(binary.BigEndian.Uint16(udp[4:6]))
	// UDP header (csum 0)
	sum += uint32(binary.BigEndian.Uint16(udp[0:2]))
	sum += uint32(binary.BigEndian.Uint16(udp[2:4]))
	sum += uint32(binary.BigEndian.Uint16(udp[4:6]))
	// Payload
	for len(payload) >= 2 {
		sum += uint32(binary.BigEndian.Uint16(payload))
		payload = payload[2:]
	}
	if len(payload) > 0 {
		sum += uint32(payload[0]) << 8
	}
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}
	cs := ^uint16(sum)
	if cs == 0 {
		cs = 0xffff
	}
	return cs
}

const ETH_P_IP = 0x0800

func openAFPacketSock() (int, error) {
	return unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, int(htons(ETH_P_IP)))
}

func ifInfo(name string) (mac [6]byte, ifindex int, autoBroadcast string, ok bool) {
	ifi, err := net.InterfaceByName(name)
	if err != nil {
		return mac, 0, "", false
	}
	copy(mac[:], ifi.HardwareAddr)
	ifindex = ifi.Index
	addrs, _ := ifi.Addrs()
	for _, a := range addrs {
		ipn, ok2 := a.(*net.IPNet)
		if !ok2 || ipn.IP == nil || ipn.IP.To4() == nil {
			continue
		}
		ip := ipn.IP.To4()
		m := ipn.Mask
		bc := net.IPv4(ip[0]|^m[0], ip[1]|^m[1], ip[2]|^m[2], ip[3]|^m[3]).String()
		return mac, ifindex, bc, true
	}
	return mac, ifindex, "", true
}

func htons(u16 uint16) uint16 { return (u16 << 8) & 0xff00 | (u16 >> 8) & 0x00ff }

func sendEthernet(fd int, ifindex int, dstMAC, srcMAC [6]byte, payload []byte) error {
	eth := make([]byte, 14+len(payload))
	copy(eth[0:6], dstMAC[:])
	copy(eth[6:12], srcMAC[:])
	binary.BigEndian.PutUint16(eth[12:14], ETH_P_IP)
	copy(eth[14:], payload)
	sa := &unix.SockaddrLinklayer{
		Protocol: htons(ETH_P_IP),
		Ifindex:  ifindex,
		Halen:    6,
		Addr:     [8]uint8{dstMAC[0], dstMAC[1], dstMAC[2], dstMAC[3], dstMAC[4], dstMAC[5]},
	}
	return unix.Sendto(fd, eth, 0, sa)
}

func main() {
	flag.Parse()

	fd, err := openAFPacketSock()
	if err != nil {
		log.Fatalf("AF_PACKET socket: %v (need root or cap_net_raw)", err)
	}
	defer unix.Close(fd)

	srcMAC, ifidx, _, ok := ifInfo(*ifaceName)
	if !ok || ifidx == 0 {
		log.Fatalf("could not read interface info for %s", *ifaceName)
	}

	var masterList []Server
	masterList, err = fetchServers()
	if err != nil {
		log.Printf("initial server list fetch: %v (continuing, will retry)", err)
	}

	refresh := time.NewTicker(30 * time.Second)
	defer refresh.Stop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, unix.SIGTERM)

	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()

	log.Printf("Binfinite Beta emulation on Linux - forging announcements on %s", *ifaceName)

	for {
		select {
		case <-refresh.C:
			masterList, err = fetchServers()
			if err != nil {
				log.Printf("refresh servers: %v", err)
			} else {
				log.Printf("Refreshed %d servers from master", len(masterList))
			}

		case <-tick.C:
			for _, server := range masterList {
				plain := buildServerPacket(server.Name)
				payload := encryptPayload(plain)
				ipudp := buildUDPPacket(server.Address, "255.255.255.255", payload)
				dstMAC := [6]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
				err := sendEthernet(fd, ifidx, dstMAC, srcMAC, ipudp)
				if err != nil {
					log.Printf("send %s to %s: %v", server.Name, server.Address, err)
				} else if *logSends {
					log.Printf("sent %s announcement to 255.255.255.255:7117", server.Name)
				}
			}

		case <-sig:
			log.Printf("exit")
			return
		}
	}
}
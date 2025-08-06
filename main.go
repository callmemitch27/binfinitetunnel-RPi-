package main

import (
	"binfinite/tunnel"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
	"unicode/utf16"
)

type Servers struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

var masterList []Servers

func populateServers() {
	for {
		resp, err := http.Get("https://binfinite.lh2.au/servers")

		if err != nil {
			log.Fatal("failed to get servers from master list")
		}

		data, err := io.ReadAll(resp.Body)

		if err != nil {
			log.Fatal("Failed to read response data from master")
		}
		err = json.Unmarshal(data, &masterList)
		if err != nil {
			log.Fatal("Failed to unmarshall response data from master", err)
		}

		log.Printf("Refereshed %d servers from master server list", len(masterList))
		time.Sleep(30 * time.Second)
	}

}

func stringToUTF16LEBytes(s string) ([]byte, error) {
	runes := []rune(s)
	utf16CodePoints := utf16.Encode(runes)

	buf := new(bytes.Buffer)
	for _, cp := range utf16CodePoints {
		err := binary.Write(buf, binary.BigEndian, cp)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// too lazy to bust out into a new cipher function for reuse
func EncryptPayload(data []byte) []byte {
	// lol
	key := []byte{0x2C, 0x0A, 0xA3, 0xD0, 0xAB, 0xB8, 0xEF, 0x33, 0x83, 0x35, 0x04, 0x02, 0xDC, 0x32, 0x8C, 0xA5}
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}

	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	encoutput := gcm.Seal(nil, nonce, data, nil)
	var result []byte

	result = append(result, nonce...)
	result = append(result, []byte{0xFF, 0x03, 0x43, 0x00}...)
	result = append(result, encoutput[len(encoutput)-16:]...) // windows ng api has crypt backwards lol
	result = append(result, encoutput[0:(len(encoutput)-16)]...)
	return result
}

func BuildServerPacket(serverName string) []byte {
	packet := []byte{0x00, 0x0d}
	name, _ := stringToUTF16LEBytes(serverName)
	//
	unkFooter := []byte{0x92, 0x87, 0x6d, 0xe1, 0xb1, 0xa5, 0xd4, 0x42, 0xbe, 0x05, 0x09, 0x94, 0x31, 0xb6, 0x37, 0xf0, 0x00, 0x00, 0x40}

	packet = append(packet, name...)
	packet = append(packet, []byte{0x00, 0x00}...)
	packet = append(packet, unkFooter...)

	return packet
}

// lol static key

func main() {

	log.Println("Binfinite Beta - https://github.com/jeffx539/binfnite")
	tun, err := tunnel.Init()

	if err != nil {
		log.Fatal("failed to create adapter", err)
	}

	log.Println("Adapter Creation OK")
	go populateServers()

	for {

		for _, server := range masterList {
			tun.Broadcast(server.Address, EncryptPayload(BuildServerPacket(server.Name)))
		}

		time.Sleep(1 * time.Second)
	}

}

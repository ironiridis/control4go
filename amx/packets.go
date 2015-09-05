package icsp

import (
	"bytes"
	"fmt"

	"github.com/ironiridis/humanhex"
)

// An Address describes AMX's 40-bit address scheme for all devices. The
// outermost ID is a System, the next innermost is a Device, and the next is a
// Port. However, these addresses are typically written D:P:S, interleaving the
// port number in the middle.
type Address struct {
	Device uint16
	Port   uint16 // wire format appears to allow 16 bits, but tools don't
	System uint16
}

func readSDPFormatAddress(b []byte) *Address {
	return &Address{
		System: (uint16(b[0]) << 8) + uint16(b[1]),
		Device: (uint16(b[2]) << 8) + uint16(b[3]),
		Port:   (uint16(b[4]) << 8) + uint16(b[5]),
	}
}

func readDPSFormatAddress(b []byte) *Address {
	return &Address{
		Device: (uint16(b[0]) << 8) + uint16(b[1]),
		Port:   (uint16(b[2]) << 8) + uint16(b[3]),
		System: (uint16(b[4]) << 8) + uint16(b[5]),
	}
}

func (dps *Address) String() string {
	return fmt.Sprintf("%d:%d:%d", dps.Device, dps.Port, dps.System)
}

//go:generate stringer -type=Msg
type Msg uint16

const (
	MsgOnlineConf   Msg = 0x0001 // no payload
	MsgOnTo         Msg = 0x0006 // DDPPSSCC (C: channel number)
	MsgOffTo        Msg = 0x0007 // DDPPSSCC (C: channel number)
	MsgLevelTo      Msg = 0x000a
	MsgStringTo     Msg = 0x000b
	MsgCommandTo    Msg = 0x000c
	MsgPressFrom    Msg = 0x0084 // DDPPSSCC
	MsgReleaseFrom  Msg = 0x0085 // DDPPSSCC
	MsgOnFrom       Msg = 0x0086 // DDPPSS??
	MsgOffFrom      Msg = 0x0087 // DDPPSS??
	MsgLevelFrom    Msg = 0x008a // DDPPSSLLTV{1+} (T: ref LevType)
	MsgStringFrom   Msg = 0x008b // DDPPSSEELs{L} (E: encoding?, s: string)
	MsgCommandFrom  Msg = 0x008c // DDPPSSEELc{L} (E: encoding?, c: command)
	MsgPortOnline   Msg = 0x0090 // DDSSPP (device, system, port)
	MsgLevelUnk1    Msg = 0x0092 // DDPPSSLL
	MsgStringLimit  Msg = 0x0093 // DDPPSSNF{N}L (F: format, 0=8bit, L: limit)
	MsgCommandLimit Msg = 0x0094 // DDPPSSNF{N}L (F: format, 0=8bit, L: limit)
	MsgLevelTypes   Msg = 0x0095 // DDPPSSLLNT{N} (T: ref LevType)
	MsgDeviceDetail Msg = 0x0097 // DDSS??OPMMHHS{16}FFVer0Mod0Manuf0ALa{L}
	// O: OID, P: PID, MM: Manuf. ID, HH: Hardware ID, S: Serial, F: Firmware ID,
	// Ver0: null-term version, Mod0: null-term model, Manuf0: null-term manuf.,
	// A: Physical address type(?), L: Phyiscal address length, a: Address bytes

	MsgTimeOfDay Msg = 0x020f // MMDDYYHHMMSSZZ (Z: timezone?)
	MsgTODAck    Msg = 0x0213 // ???? (no clue; saw \x02\x0f\x01\xa8 once)
	MsgPing      Msg = 0x0501 // DDSS
	MsgHeartbeat Msg = 0x0502 // ?GMDYYHMS???DateString0 (G: alternating 0/1)
	MsgPong      Msg = 0x0581 // DDSSPPHHALa{L} H: Hardware ID
	// A: Physical address type(?), L: Phyiscal address length, a: Address bytes

	MsgAuthUnk1 Msg = 0x0701
	MsgAuthUnk2 Msg = 0x0702
	MsgAuthUnk3 Msg = 0x0703 // observed payloads:
	// success? \x00\x01\x00\x10
	// proceeding encrypted? \x00\x03\x00\x18
	// so it could be a bitmask, ABCD, where
	// B & 0x02 or D & 0x08 == encryption
	// or one of those could indicate an encryption method.
	// not going to push too hard down that road until we have a more robust (aka
	// working) implementation.
)

type LevType uint8

const (
	LevTypeSInt LevType = 0x21
)

// Packet is an interface for getting the information out of an ICSP Packet.
type Packet interface {
	Parse() error
	String() string
}

// PacketParser buffers responses and reads/parses them in order.
type PacketParser struct {
	buf *bytes.Buffer
	ch  chan Packet
}

func byteSlice8bitSum(b []byte) (s byte) {
	for _, v := range b {
		s = (s + v) & 0xff
	}
	return
}

// Parse will read as many packets as possible, outputting them on the channel.
func (s *PacketParser) Parse() (parsed int) {
	for {
		if s.buf.Len() < 3 {
			return
		}
		b := s.buf.Bytes()
		if b[0] != 0x02 && b[0] != 0x04 {
			// observed packets always begin with 0x02, assume a desync
			fmt.Printf("desync; truncating with buf: %s\n", humanhex.String(b, 3))
			s.buf.Reset()
			return
		}
		l := (int(b[1]) << 8) + int(b[2])

		if s.buf.Len() < (l + 4) {
			return
		}

		// header is 3 bytes, plus 1 byte checksum
		d := make([]byte, l+4)
		s.buf.Read(d)
		s.ch <- NewRawPacket(d)

		parsed++
	}
}

// RawPacket is an uninterpreted packet that can be further processed.
type RawPacket struct {
	raw           []byte
	encrypted     bool
	checksumValid bool
	unk1          uint8 // byte 3, observed to be 0x02 usually
	flags         uint8 //
	To            *Address
	From          *Address
	unk2          uint8 // byte 17, observed to be 0x0f usually
	unk3          uint8 // byte 18, observed to be 0x03 or 0xff
	seq           uint8
	Kind          Msg
	Payload       []byte
}

func NewRawPacket(d []byte) *RawPacket {
	p := &RawPacket{raw: d}
	switch d[0] {
	case 0x02:
		{
			p.encrypted = false
		}
	case 0x04:
		{
			p.encrypted = true
		}
	}
	p.unk1 = uint8(d[3])
	p.flags = uint8(d[4])
	p.To = readSDPFormatAddress(d[5:])
	p.From = readSDPFormatAddress(d[11:])
	p.unk2 = uint8(d[17])
	p.unk3 = uint8(d[18])
	p.seq = uint8(d[19])
	p.Kind = Msg((uint16(d[20]) << 8) + uint16(d[21]))
	p.Payload = d[22 : len(d)-1]

	p.checksumValid = (byteSlice8bitSum(d[:len(d)-1]) == d[len(d)-1])

	if !p.checksumValid {
		fmt.Println("packet checksum invalid")
	}
	if p.unk1 != 0x02 {
		fmt.Println("packet invariant unk1 different")
	}
	if p.unk2 != 0x0f {
		fmt.Println("packet invariant unk2 different")
	}

	return p
}

// Write will add data to the parsing queue, staged for a call to Parse.
func (s *PacketParser) Write(b []byte) (int, error) {
	return s.buf.Write(b)
}

// NewPacketParser returns an instance of a parser with a channel for receiving
// complete packets.
func NewPacketParser() (*PacketParser, chan Packet) {
	ch := make(chan Packet)
	j := &PacketParser{buf: new(bytes.Buffer), ch: ch}
	return j, ch
}

// RawPayload returns the byte data of the packet
func (p *RawPacket) RawPayload() []byte {
	return p.raw
}

// String returns a dump of the packet data
func (p *RawPacket) String() string {
	return fmt.Sprintf("%v -> %v %v %s",
		p.From, p.To, p.Kind, humanhex.String(p.Payload, 3))
}

// Parse is currently a no-op.
func (p *RawPacket) Parse() error { return nil }

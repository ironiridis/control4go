package cip

import (
	"bytes"
	"fmt"
)

// Packet is an interface for getting the information out of a CIP Packet.
type Packet interface {
	Parse() error
	String() string
	RawPayload() []byte
}

// GreetPacket is the initial packet a processor sends on connection.
type GreetPacket struct {
	raw []byte
}

// GreetPacketResponse is the packet a device sends in response to a GreetPacket.
type GreetPacketResponse struct {
	raw []byte
	// IPID is the system-unique number of the device connecting to the processor.
	// Crestron defines the range as 0x03 to 0xFE.
	IPID byte
}

// SetPacket is sent to communicate a change in state (press, release, etc).
type SetPacket struct {
	raw []byte
}

// EchoRequestPacket requests that the peer prove the connection is still good.
type EchoRequestPacket struct {
	raw []byte
}

// EchoResponsePacket is a response to EchoRequestPacket.
type EchoResponsePacket struct {
	raw []byte
}

//go:generate stringer -type=cipPacketType
type cipPacketType byte

const (
	packetGreetResponse cipPacketType = 0x01
	packetSet           cipPacketType = 0x05
	packetEchoRequest   cipPacketType = 0x0d
	packetEchoResponse  cipPacketType = 0x0e
	packetGreet         cipPacketType = 0x0f
)

// RawPacket is an uninterpreted packet that can be further processed.
type RawPacket struct {
	raw           []byte
	kind          cipPacketType
	payloadLength int
}

// PacketParser buffers responses and reads/parses them in order.
type PacketParser struct {
	buf *bytes.Buffer
	ch  chan Packet
}

// Parse will read as many packets as possible, outputting them on the channel.
func (s *PacketParser) Parse() (parsed int) {
	for {
		if s.buf.Len() < 3 {
			return
		}
		b := s.buf.Bytes()
		l := 2 + (int(b[1]) << 8) + int(b[2])

		if s.buf.Len() < l {
			return
		}
		d := make([]byte, l+1)
		s.buf.Read(d)
		s.ch <- &RawPacket{raw: d}

		parsed++
	}
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

// Parse reads the type of packet and the payload length.
func (p *RawPacket) Parse() error {
	p.kind = cipPacketType(p.raw[0])
	p.payloadLength = (int(p.raw[1]) << 8) + int(p.raw[2])
	switch p.kind {
	case packetGreet:

	}
	return nil
}

// String returns a dump of the packet data
func (p *RawPacket) String() string {
	err := p.Parse()
	if err != nil {
		return "<error parsing>"
	}
	return fmt.Sprintf("%v raw=% 2x", p.kind, p.RawPayload())
}

// RawPayload returns the byte data of the packet
func (p *RawPacket) RawPayload() []byte {
	if p.payloadLength == 0 {
		return nil
	}
	return p.raw[3 : 3+p.payloadLength]
}

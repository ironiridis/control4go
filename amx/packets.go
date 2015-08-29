package icsp

import (
	"bytes"
	"fmt"
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

// RawPacket is an uninterpreted packet that can be further processed.
type RawPacket struct {
	raw           []byte
	payloadLength int
}

// Parse is currently a no-op
func (s *PacketParser) Parse() int { return 0 }

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
	if p.payloadLength == 0 {
		return nil
	}
	return p.raw
}

// String returns a dump of the packet data
func (p *RawPacket) String() string {
	return fmt.Sprintf("raw=% 2x", p.RawPayload())
}

package icsp

import (
	"bytes"
	"fmt"

	"github.com/ironiridis/humanhex"
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

// Parse will read as many packets as possible, outputting them on the channel.
func (s *PacketParser) Parse() (parsed int) {
	for {
		if s.buf.Len() < 3 {
			return
		}
		b := s.buf.Bytes()
		if b[0] != 0x02 {
			// observed packets always begin with 0x02, assume a desync
			s.buf.Reset()
			return
		}
		// header is 3 bytes, plus length of packet
		l := 3 + (int(b[1]) << 8) + int(b[2])

		if s.buf.Len() < l {
			return
		}
		d := make([]byte, l+1)
		s.buf.Read(d)

		rp := &RawPacket{raw: d, payloadLength: l}
		s.ch <- rp

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

// RawPayload returns the byte data of the packet
func (p *RawPacket) RawPayload() []byte {
	if p.payloadLength == 0 {
		return nil
	}
	return p.raw
}

// String returns a dump of the packet data
func (p *RawPacket) String() string {
	return fmt.Sprintf("raw=%s", humanhex.String(p.RawPayload(), 2))
}

// Parse is currently a no-op.
func (p *RawPacket) Parse() error { return nil }

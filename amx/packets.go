package icsp

import "fmt"

// Packet is an interface for getting the information out of a CIP Packet.
type Packet interface {
	Parse() error
	String() string
}

type PacketParser struct{}
type RawPacket struct {
	raw           []byte
	payloadLength int
}

func (p *PacketParser) Write(b []byte) (int, error) { return 0, nil }
func (p *PacketParser) Parse() int                  { return 0 }

func NewPacketParser() (*PacketParser, chan Packet) { return nil, nil }

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

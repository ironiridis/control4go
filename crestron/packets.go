package cip

import "fmt"

// Packet is an interface for getting the information out of a CIP Packet.
type Packet interface {
	Parse() error
	String() string
}

// GreetPacket is the initial packet a processor sends on connection.
type GreetPacket struct {
	raw []byte
}

// Parse currently is a no-op.
func (p *GreetPacket) Parse() error {
	// no-op
	return nil
}

// String returns a string representation of the packet.
func (p *GreetPacket) String() string {
	return fmt.Sprintf("Greeting Packet (p=% 2x)", p.raw)
}

// GreetPacketResponse is the packet a device sends in response to a GreetPacket.
type GreetPacketResponse struct {
	raw []byte
	// IPID is the system-unique number of the device connecting to the processor.
	// Crestron defines the range as 0x03 to 0xFE.
	IPID byte
}

// Parse reads the IPID from the packet.
func (p *GreetPacketResponse) Parse() error {
	p.IPID = p.raw[5]
	if p.IPID < 0x03 || p.IPID > 0xFE {
		return fmt.Errorf("IPID outside of valid range: %d", p.IPID)
	}
	return nil
}

func (p *GreetPacketResponse) String() string {
	return fmt.Sprintf("Greeting Reponse Packet w/ IPID=%d (p=% 2x)", p.IPID, p.raw)
}

// SetPacket is sent to communicate a change in state (press, release, etc).
type SetPacket struct {
	raw []byte
}

// Parse currently is a no-op.
func (p *SetPacket) Parse() error {
	// no-op
	return nil
}

func (p *SetPacket) String() string {
	return fmt.Sprintf("Set Packet (p=% 2x)", p.raw)
}

// EchoRequestPacket requests that the peer prove the connection is still good.
type EchoRequestPacket struct {
	raw []byte
}

// Parse currently is a no-op.
func (p *EchoRequestPacket) Parse() error {
	// no-op
	return nil
}

func (p *EchoRequestPacket) String() string {
	return fmt.Sprintf("Echo Request Packet (p=% 2x)", p.raw)
}

// EchoResponsePacket is a response to EchoRequestPacket.
type EchoResponsePacket struct {
	raw []byte
}

// Parse currently is a no-op.
func (p *EchoResponsePacket) Parse() error {
	// no-op
	return nil
}

func (p *EchoResponsePacket) String() string {
	return fmt.Sprintf("Echo Response Packet (p=% 2x)", p.raw)
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

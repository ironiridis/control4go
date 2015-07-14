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
	return fmt.Sprintf("Greeting Reponse Packet w/ IPID=%02x (p=% 2x)", p.IPID, p.raw)
}

// SetPacket is sent to communicate a change in state (press, release, etc).
type SetPacket struct {
	raw        []byte
	JoinNumber uint16
	SetType    int // digital, analog (TODO make consts)
	Value      uint16
}

// Parse a Set packet and fills struct fields from that data.
func (p *SetPacket) Parse() error {
	switch p.raw[2] {
	case 0x03: // Digital set
		p.SetType = 1
		// Not a typo: join number bytes are reversed depending on set type
		p.JoinNumber = 1 + (uint16(p.raw[5]&0x7f) << 8) + (uint16(p.raw[4]))
		if (uint16(p.raw[5] & 0x80)) == 0 { // 1=press, 0=release
			p.Value = 1
		} else {
			p.Value = 0
		}
	case 0x05: // Analog set
		// 00 00 05 XX jj jj vv vv
		// It's not clear what XX is; might be a sample rate? (is usually 14)
		p.SetType = 2
		// Not a typo: join number bytes are reversed depending on set type
		p.JoinNumber = 1 + (uint16(p.raw[4]) << 8) + (uint16(p.raw[5]))
		p.Value = (uint16(p.raw[6]) << 8) + uint16(p.raw[7])
	}
	return nil
}

func (p *SetPacket) String() string {
	switch p.SetType {
	case 1:
		if p.Value > 0 {
			return fmt.Sprintf("Set Packet: Press on join %d (p=% 2x)", p.JoinNumber, p.raw)
		}
		return fmt.Sprintf("Set Packet: Release on join %d (p=% 2x)", p.JoinNumber, p.raw)
	case 2:
		return fmt.Sprintf("Set Packet: Analog value %d on join %d (p=% 2x)", p.Value, p.JoinNumber, p.raw)
	}
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

// Promote turns a raw packet into a fully parsed packet with specific type.
func (p *RawPacket) Promote() (Packet, error) {
	var n Packet
	switch p.kind {
	case packetGreet:
		n = &GreetPacket{raw: p.RawPayload()}
	case packetGreetResponse:
		n = &GreetPacketResponse{raw: p.RawPayload()}
	case packetSet:
		n = &SetPacket{raw: p.RawPayload()}
	case packetEchoRequest:
		n = &EchoRequestPacket{raw: p.RawPayload()}
	case packetEchoResponse:
		n = &EchoResponsePacket{raw: p.RawPayload()}
	default:
		return p, nil
	}
	err := n.Parse()
	return n, err
}

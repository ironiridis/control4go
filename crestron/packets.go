package cip

import "fmt"
import "unicode/utf16"

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
	Type       SetPacketType
	Value      uint16
}

// SetPacketType specifies the kind of "set" operation is in this packet.
type SetPacketType int

const (
	unknownSetPacketType SetPacketType = iota
	// DigitalTransition refers to a press/release or a high/low state change.
	DigitalTransition
	// AnalogUpdate refers to an analog value change.
	AnalogUpdate
	// SerialTraffic refers to a packet describing serial traffic.
	SerialTraffic
)

// Parse a Set packet and fills struct fields from that data.
func (p *SetPacket) Parse() error {
	switch p.raw[3] {
	case 0x27: // Digital set
		p.Type = DigitalTransition
		// Not a typo: join number bytes are reversed depending on set type
		p.JoinNumber = 1 + (uint16(p.raw[5]&0x7f) << 8) + (uint16(p.raw[4]))
		if (uint16(p.raw[5] & 0x80)) == 0 {
			p.Value = 1 // high/press
		} else {
			p.Value = 0 // low/release
		}
	case 0x14: // Analog set
		p.Type = AnalogUpdate
		// Not a typo: join number bytes are reversed depending on set type
		p.JoinNumber = 1 + (uint16(p.raw[4]) << 8) + (uint16(p.raw[5]))
		p.Value = (uint16(p.raw[6]) << 8) + uint16(p.raw[7])
	case 0x02: // Serial traffic
		p.Type = SerialTraffic

	}

	return nil
}

func (p *SetPacket) String() string {
	switch p.Type {
	case DigitalTransition:
		if p.Value > 0 {
			return fmt.Sprintf("Set Packet: Press on join %d (p=% 2x)", p.JoinNumber, p.raw)
		}
		return fmt.Sprintf("Set Packet: Release on join %d (p=% 2x)", p.JoinNumber, p.raw)
	case AnalogUpdate:
		return fmt.Sprintf("Set Packet: Analog value %d on join %d (p=% 2x)", p.Value, p.JoinNumber, p.raw)
	case SerialTraffic:
		return fmt.Sprintf("Set Packet: Serial traffic [can contain multiple transitions] (p=% 2x)", p.raw)
	}
	return fmt.Sprintf("Set Packet [unknown specific type] (p=% 2x)", p.raw)
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

// SerialDataPacket is an update consisting of serial traffic.
type SerialDataPacket struct {
	raw        []byte
	JoinNumber uint16
	Value      string
	Encoding   int
}

func (p *SerialDataPacket) Parse() error {
	l := (uint32(p.raw[0]) << 24) + (uint32(p.raw[1]) << 16) + (uint32(p.raw[2]) << 8) + uint32(p.raw[3])
	p.JoinNumber = 1 + (uint16(p.raw[5]) << 8) + (uint16(p.raw[6]))
	p.Encoding = int(p.raw[7])
	switch p.Encoding {
	case 3: // ASCII
		p.Value = string(p.raw[8 : 4+l])
	case 7: // UTF-16
		decbuf := make([]uint16, 0, (l-4)/2)
		g := uint32(0)
		for g < ((l - 4) / 2) {
			decbuf = append(decbuf, uint16(p.raw[(g*2)+8])+(uint16(p.raw[(g*2)+9])<<8))
			g++
		}
		p.Value = string(utf16.Decode(decbuf))
	}
	return nil
}

func (p *SerialDataPacket) String() string {
	return fmt.Sprintf("Serial Data Packet: Value=%q on join %d (p=% 2x)", p.Value, p.JoinNumber, p.raw)
}

//go:generate stringer -type=cipPacketType
type cipPacketType byte

const (
	packetGreetResponse cipPacketType = 0x01
	packetSet           cipPacketType = 0x05
	packetEchoRequest   cipPacketType = 0x0d
	packetEchoResponse  cipPacketType = 0x0e
	packetGreet         cipPacketType = 0x0f
	packetSerialData    cipPacketType = 0x12
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
	case packetSerialData:
		n = &SerialDataPacket{raw: p.RawPayload()}
	default:
		return p, nil
	}
	err := n.Parse()
	return n, err
}

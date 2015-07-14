package cip

import "bytes"

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

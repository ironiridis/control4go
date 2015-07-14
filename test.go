package main

import (
	"bytes"
	"fmt"
	"net"
)

//go:generate stringer -type=cipPacketType
type cipPacketType int

const (
	// PacketSet appears to be used to set a value
	PacketSet cipPacketType = 0x05
	// PacketEchoRequest is a keep-alive request
	PacketEchoRequest cipPacketType = 0x0d
	// PacketEchoResponse is a keep-alive response
	PacketEchoResponse cipPacketType = 0x0e
)

type cipPacketStream struct {
	buf *bytes.Buffer
	src string
	ch  chan *cipPacket
}

type cipPacket struct {
	raw           []byte
	kind          cipPacketType
	payloadLength int
	parsed        bool
	src           string
}

func (s *cipPacketStream) Parse() (parsed int) {
	for {
		if s.buf.Len() == 0 {
			return
		}
		if s.buf.Len() < 3 {
			//fmt.Printf("%s: Stream is too short to determine length, have %d\n", s.src, s.buf.Len())
			return
		}
		b := s.buf.Bytes()
		l := 2 + (int(b[1]) << 8) + int(b[2])
		//l := 2 + int(b[2])

		if s.buf.Len() < l {
			//fmt.Printf("%s: Stream is too short to complete packet, need: %d, have %d\n", s.src, l, s.buf.Len())
			return
		}
		d := make([]byte, l+1)
		s.buf.Read(d)
		s.ch <- &cipPacket{raw: d, src: s.src}
		parsed++
	}
}

func (p *cipPacket) Parse() bool {
	p.kind = cipPacketType(p.raw[0])
	p.payloadLength = (int(p.raw[1]) << 8) + int(p.raw[2])

	p.parsed = true
	return p.parsed
}

func (p *cipPacket) Dump() {
	fmt.Printf("%s: len=%d, raw=%x\n", p.src, len(p.raw), p.raw)
	if !p.parsed {
		if !p.Parse() {
			fmt.Printf("%s: Failed to parse packet.\n", p.src)
			return
		}
	}
	fmt.Printf("%s: %s (payload=%d)\n", p.src, p.kind.String(), p.payloadLength)
}

func pipeIntercept(a, b net.Conn, done chan bool, stream *cipPacketStream) {
	buf := make([]byte, 1500)
	for {
		readN, err := a.Read(buf)
		if err != nil {
			fmt.Printf("%s: read returned error (%v)\n", stream.src, err)
			break
		}

		// intercept and push to parser
		stream.buf.Write(buf[:readN])
		stream.Parse()

		writeN, err := b.Write(buf[:readN])
		if writeN < readN {
			fmt.Printf("%s: write returned too few bytes (%d, not %d)\n", stream.src, writeN, readN)
			break
		}
		if err != nil {
			fmt.Printf("%s: write returned error (%v)\n", stream.src, err)
			break
		}
	}
	done <- true
}

func main() {
	pipeDone := make(chan bool)
	pkt := make(chan *cipPacket)
	for {
		panelListener, err := net.Listen("tcp", ":41794")
		if err != nil {
			panic(err)
		}
		panel, err := panelListener.Accept()
		if err != nil {
			panic(err)
		}
		panelListener.Close()
		processor, err := net.Dial("tcp", "10.0.9.233:41794")
		if err != nil {
			panic(err)
		}
		fromProc := cipPacketStream{buf: new(bytes.Buffer), src: "Processor", ch: pkt}
		fromPanel := cipPacketStream{buf: new(bytes.Buffer), src: "Panel", ch: pkt}
		go pipeIntercept(processor, panel, pipeDone, &fromProc)
		go pipeIntercept(panel, processor, pipeDone, &fromPanel)
		var p *cipPacket
	ReadCopyLoop:
		for {
			select {
			case <-pipeDone:
				fmt.Printf("Connection closed\n")
				break ReadCopyLoop
			case p = <-pkt:
				p.Dump()
			}
		}
		panel.Close()
		processor.Close()
	}
}

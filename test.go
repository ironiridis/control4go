package main

import (
	"bytes"
	"fmt"
	"net"
)

type cipPacketType int

const (
	unk00 cipPacketType = iota
	unk01
	unk02
	unk03
	unk04
	unk05
	unk06
	unk07
	unk08
	unk09
	unk10
	unk11
	unk12
	unk13
	unk14
	unk15
	unk16
	unk17
	unk18
	unk19
	unk1a
	unk1b
	unk1c
	unk1d
	unk1e
	unk1f
)

type cipPacket struct {
	raw  bytes.Buffer
	kind cipPacketType
}

func intercept(pkt []byte, direction int) {

}

func pipeIntercept(a, b net.Conn, done chan bool, direction int) {
	buf := make([]byte, 1500)
	for {
		readN, err := a.Read(buf)
		if err != nil {
			fmt.Printf("%d: read returned error (%v)\n", direction, err)
			break
		}
		writeN, err := b.Write(buf[:readN-1])
		if writeN < readN {
			fmt.Printf("%d: write returned too few bytes (%d, not %d)\n", direction, writeN, readN)
			break
		}
		if err != nil {
			fmt.Printf("%d: write returned error (%v)\n", direction, err)
			break
		}
	}
	done <- true
}

func main() {
	pipeDone := make(chan bool)
	panelListener, err := net.Listen("tcp", ":41794")
	if err != nil {
		panic(err)
	}
	for {
		panel, err := panelListener.Accept()
		if err != nil {
			panic(err)
		}
		processor, err := net.Dial("tcp", "10.0.9.233:41794")
		if err != nil {
			panic(err)
		}
		go pipeIntercept(processor, panel, pipeDone, 0)
		go pipeIntercept(panel, processor, pipeDone, 1)
		<-pipeDone
		panel.Close()
		processor.Close()
	}
}

package main

import (
	"fmt"
	"net"

	"github.com/ironiridis/control4go/amx"
)

func pipeIntercept(a, b net.Conn, done chan bool, stream *icsp.PacketParser) {
	buf := make([]byte, 1500)
	for {
		readN, err := a.Read(buf)
		if err != nil {
			fmt.Printf("read returned error (%v)\n", err)
			break
		}

		// intercept and push to parser
		stream.Write(buf[:readN])
		stream.Parse()

		writeN, err := b.Write(buf[:readN])
		if writeN < readN {
			fmt.Printf("write returned too few bytes (%d, not %d)\n", writeN, readN)
			break
		}
		if err != nil {
			fmt.Printf("write returned error (%v)\n", err)
			break
		}
	}
	select {
	case done <- true: // signal completion (usually disconnect)
	default: // channel is closed; connection is already tearing down
	}
}

func main() {
	for {
		panelListener, err := net.Listen("tcp", ":1319")
		if err != nil {
			panic(err)
		}
		panel, err := panelListener.Accept()
		if err != nil {
			panic(err)
		}
		panelListener.Close()
		processor, err := net.Dial("tcp", "192.168.0.26:1319")
		if err != nil {
			panic(err)
		}
		fromProc, pktFromProc := icsp.NewPacketParser()
		fromDevice, pktFromDevice := icsp.NewPacketParser()
		pipeDone := make(chan bool, 2)
		go pipeIntercept(processor, panel, pipeDone, fromProc)
		go pipeIntercept(panel, processor, pipeDone, fromDevice)
		var p icsp.Packet

	ReadCopyLoop:
		for {
			select {
			case <-pipeDone:
				fmt.Printf("Connection closed\n")
				break ReadCopyLoop
			case p = <-pktFromProc:
				fmt.Printf("From Processor | %v\n", p.String())
			case p = <-pktFromDevice:
				fmt.Printf("From Device    | %v\n", p.String())
			}
		}

		panel.Close()
		processor.Close()
	}
}

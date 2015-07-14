package main

import (
	"fmt"
	"net"

	"github.com/ironiridis/control4go/crestron"
)

func pipeIntercept(a, b net.Conn, done chan bool, stream *cip.PacketParser) {
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
	done <- true
}

func main() {
	pipeDone := make(chan bool)
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
		fromProc, pktFromProc := cip.NewPacketParser()
		fromDevice, pktFromDevice := cip.NewPacketParser()
		go pipeIntercept(processor, panel, pipeDone, fromProc)
		go pipeIntercept(panel, processor, pipeDone, fromDevice)
		var p cip.Packet

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

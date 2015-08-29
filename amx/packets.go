package icsp

type PacketParser struct{}
type Packet struct{}

func (p *PacketParser) Write(b []byte) (int, error) { return 0, nil }
func (p *PacketParser) Parse() int                  { return 0 }
func (p *Packet) String() string                    { return "" }

func NewPacketParser() (*PacketParser, chan Packet) { return nil, nil }

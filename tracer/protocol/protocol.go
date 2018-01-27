package protocol

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"

	"github.com/yuuki0xff/xtcp"
)

const (
	ProtocolVersion = "1"
)

// isCompatibleVersion returns true if "version" has compatibility of current version
func isCompatibleVersion(version string) bool {
	return ProtocolVersion == version
}

// Message: [size int32] [hp HeaderPacket] [p xtcp.Packet]
// size =  hp.size() + p.size()
type Proto struct{}

func (pr Proto) PackSize(p xtcp.Packet) int {
	b, err := pr.Pack(p)
	if err != nil {
		log.Panic(err)
	}
	return len(b)
}
func (pr Proto) PackTo(p xtcp.Packet, w io.Writer) (int, error) {
	b, err := pr.Pack(p)
	if err != nil {
		return 0, err
	}
	return w.Write(b)
}
func (pr Proto) Pack(p xtcp.Packet) ([]byte, error) {
	var hp HeaderPacket
	var buf bytes.Buffer

	// prepare header packet
	hp.PacketType = detectPacketType(p)

	// ensure uint32 space
	buf.WriteByte(0)
	buf.WriteByte(0)
	buf.WriteByte(0)
	buf.WriteByte(0)

	// build buf
	if err := marshalPacket(&hp, &buf); err != nil {
		return nil, err
	}
	if err := marshalPacket(p, &buf); err != nil {
		return nil, err
	}

	// write data size
	b := buf.Bytes()
	packetSize := uint32(len(b) - 4)
	binary.BigEndian.PutUint32(b[:4], packetSize)
	return b, nil
}
func (pr Proto) Unpack(b []byte) (xtcp.Packet, int, error) {
	var hp HeaderPacket

	if len(b) < 4 {
		// buf size not enough for unpack
		return nil, 0, nil
	}
	dataPacketSizeBin := b[:4]
	dataPacketSize := int(binary.BigEndian.Uint32(dataPacketSizeBin))
	packetSize := 4 + dataPacketSize
	packetData := b[4:]
	if len(packetData) < dataPacketSize {
		// buf size not enough for unpack
		return nil, 0, nil
	}

	buf := bytes.NewBuffer(packetData)
	if err := hp.Unmarshal(buf); err != nil {
		return nil, packetSize, err
	}

	p := createPacket(PacketType(hp.PacketType))
	if err := unmarshalPacket(p, buf); err != nil {
		return nil, packetSize, err
	}
	return p, packetSize, nil
}

func marshalPacket(p xtcp.Packet, buf io.Writer) error {
	m := p.(Marshalable)
	return m.Marshal(buf)
}

func unmarshalPacket(p xtcp.Packet, r io.Reader) error {
	m := p.(Marshalable)
	return m.Unmarshal(r)
}

package protocol

import (
	"io"

	"bytes"
	"encoding/gob"
	"log"

	"encoding/binary"

	"github.com/xfxdev/xtcp"
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
	log.Printf("DEBUG: Proto.PackSize(): p=%+v size=%d", p, len(b))
	return len(b)
}
func (pr Proto) PackTo(p xtcp.Packet, w io.Writer) (int, error) {
	b, err := pr.Pack(p)
	if err != nil {
		return 0, err
	}
	log.Printf("DEBUG: Proto.PackTo(): p=%+v, len(b)=%d b=%+v", p, len(b), b)
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
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(&hp); err != nil {
		return nil, err
	}
	if err := enc.Encode(p); err != nil {
		return nil, err
	}

	// write data size
	b := buf.Bytes()
	packetSize := uint32(len(b) - 4)
	binary.BigEndian.PutUint32(b[:4], packetSize)
	log.Printf("DEBUG: Proto.Pack(): p=%+v len(b)=%d b=%+v", p, len(b), b)
	return b, nil
}
func (pr Proto) Unpack(b []byte) (xtcp.Packet, int, error) {
	log.Printf("DEBUG: Proto.Unpack(): len(b)=%d b=%+v", len(b), b)
	var hp HeaderPacket
	var buf bytes.Buffer

	if len(b) < 4 {
		// buf size not enough for unpack
		log.Printf("DEBUG: Proto.Unpack(): requireSize>=4 p=%+v size=%d, err=%+v", nil, 0, nil)
		return nil, 0, nil
	}
	packetSizeBin := b[:4]
	packetSize := int(binary.BigEndian.Uint32(packetSizeBin))
	packetData := b[4:]
	if len(packetData) < packetSize {
		// buf size not enough for unpack
		log.Printf("DEBUG: Proto.Unpack(): requireSize>=4+%d packetSize=%d p=%+v size=%d, err=%+v", packetSize, packetSize, nil, 0, nil)
		return nil, 0, nil
	}

	buf.Write(packetData)
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&hp); err != nil {
		log.Printf("DEBUG: Proto.Unpack(): p=%+v size=%d, err=%+v", nil, packetSize, err)
		return nil, packetSize, err
	}

	p := createPacket(hp.PacketType)
	err := dec.Decode(p)
	if err != nil {
		return nil, packetSize, err
	}
	log.Printf("DEBUG: Proto.Unpack(): p=%+v size=%d, err=%+v", p, packetSize, nil)
	return p, packetSize, nil
}

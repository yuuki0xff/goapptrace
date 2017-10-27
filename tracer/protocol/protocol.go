package protocol

import (
	"io"

	"bytes"
	"encoding/gob"
	"log"

	"encoding/binary"

	"github.com/xfxdev/xtcp"
)

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
	return b, nil
}
func (pr Proto) Unpack(b []byte) (xtcp.Packet, int, error) {
	var hp HeaderPacket
	var buf bytes.Buffer

	if len(b) < 4 {
		// buf size not enough for unpack
		return nil, 0, nil
	}
	packetData := b[4:]
	packetSize := int(binary.BigEndian.Uint32(packetData))
	if len(packetData) < packetSize {
		// buf size not enough for unpack
		return nil, 0, nil
	}

	buf.Write(packetData)
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&hp); err != nil {
		return nil, packetSize, err
	}

	p := createPacket(hp.PacketType)
	err := dec.Decode(p)
	if err != nil {
		return nil, packetSize, err
	}
	return p, packetSize, nil
}

// detectPacketType returns PacketType of packet.
// If packet is not PacketType, will be occurs panic.
func detectPacketType(packet xtcp.Packet) PacketType {
	switch packet.(type) {
	case LogPacket:
		return LogPacketType
	case PingPacket:
		return PingPacketType
	case ShutdownPacket:
		return ShutdownPacketType
	case StartTraceCmdPacket:
		return StartTraceCmdPacketType
	case StopTraceCmdPacket:
		return StopTraceCmdPacketType
	default:
		log.Panic("bug")
		return UnknownPacketType
	}
}

// createPacket returns empty packet.
func createPacket(packetType PacketType) xtcp.Packet {
	switch packetType {
	case LogPacketType:
		return &LogPacket{}
	case PingPacketType:
		return &PingPacket{}
	case ShutdownPacketType:
		return &PingPacket{}
	case StartTraceCmdPacketType:
		return &StartTraceCmdPacket{}
	case StopTraceCmdPacketType:
		return &StopTraceCmdPacket{}
	default:
		log.Panic("bug")
		return nil
	}
}

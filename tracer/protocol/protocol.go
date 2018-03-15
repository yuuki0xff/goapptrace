package protocol

import (
	"encoding/binary"
	"io"
	"log"

	. "github.com/yuuki0xff/goapptrace/tracer/util"
	"github.com/yuuki0xff/xtcp"
)

const (
	ProtocolVersion = "1"

	// パケットをエンコードすることにより増加するバイト数。
	// 内約は、パケットサイズ(4byte)+HeaderPacket(1byte)
	PacketHeaderSize = 5
)

// isCompatibleVersion returns true if "version" has compatibility of current version
func isCompatibleVersion(version string) bool {
	return ProtocolVersion == version
}

type ProtoInterface interface {
	xtcp.Protocol
	PackToByteSlice(p xtcp.Packet, buf []byte) int64
}

// Message: [size int32] [hp HeaderPacket] [p xtcp.Packet]
// size =  hp.size() + p.size()
type Proto struct{}

// このメソッドはどこからも使用されていないため、実装していない。
// もしこのメソッドを呼び出すと、panicする。
func (pr Proto) PackSize(p xtcp.Packet) int {
	log.Panic("this method is not implemented")
	panic(nil)
}
func (pr Proto) PackTo(p xtcp.Packet, w io.Writer) (int, error) {
	if mergePkt, ok := p.(*MergePacket); ok {
		// MergePacketは、シリアライズ済みデータをストリームに書き込むだけで良い。
		n, err := mergePkt.WriteTo(w)
		return int(n), err
	}
	buf := make([]byte, 1024)
	n := pr.PackToByteSlice(p, buf)
	return w.Write(buf[:n])
}
func (pr Proto) PackToByteSlice(p xtcp.Packet, buf []byte) int64 {
	// prepare header packet
	hp := HeaderPacket{
		PacketType: detectPacketType(p),
	}

	// build buf
	payloadBuf := buf[4:]
	headerSize := marshalPacket(&hp, payloadBuf)
	body := payloadBuf[headerSize:]
	bodySize := marshalPacket(p, body)

	// write payload size
	payloadSize := uint32(headerSize + bodySize)
	binary.BigEndian.PutUint32(buf[:4], payloadSize)

	// write to connection
	packetSize := payloadSize + 4
	return int64(packetSize)
}

// このメソッドはどこからも使用されていないため、実装していない。
// もしこのメソッドを呼び出すと、panicする。
func (pr Proto) Pack(p xtcp.Packet) ([]byte, error) {
	log.Panic("this method is not implemented")
	panic(nil)
}
func (pr Proto) Unpack(b []byte) (xtcp.Packet, int, error) {
	var hp HeaderPacket
	var p xtcp.Packet

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

	if err := PanicHandler(func() {
		n := hp.Unmarshal(packetData)
		p = createPacket(hp.PacketType)
		unmarshalPacket(p, packetData[n:])
	}); err != nil {
		return nil, packetSize, err
	}
	return p, packetSize, nil
}

func marshalPacket(p xtcp.Packet, buf []byte) int64 {
	m := p.(Marshalable)
	return m.Marshal(buf)
}

func unmarshalPacket(p xtcp.Packet, buf []byte) int64 {
	m := p.(Marshalable)
	return m.Unmarshal(buf)
}

package protocol

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"sync"

	. "github.com/yuuki0xff/goapptrace/tracer/util"
	"github.com/yuuki0xff/xtcp"
)

const (
	ProtocolVersion = "1"

	DefaultPacketBufferSize = 1024
)

var (
	bufferPool = sync.Pool{
		New: func() interface{} {
			return bytes.NewBuffer(make([]byte, DefaultPacketBufferSize))
		},
	}
)

// isCompatibleVersion returns true if "version" has compatibility of current version
func isCompatibleVersion(version string) bool {
	return ProtocolVersion == version
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

	buf := bufferPool.Get().(*bytes.Buffer)
	// deferのオーバーヘッドを削減するため、bufferPool.Put(buf)は関数の末尾で行う。
	// この関数の実行中にpanicすると、このbufはpoolに戻されない可能性がある。
	buf.Reset()

	// prepare header packet
	hp := HeaderPacket{
		PacketType: detectPacketType(p),
	}

	// ensure uint32 space
	buf.WriteByte(0)
	buf.WriteByte(0)
	buf.WriteByte(0)
	buf.WriteByte(0)

	// build buf
	if err := PanicHandler(func() {
		marshalPacket(&hp, buf)
		marshalPacket(p, buf)
	}); err != nil {
		return 0, err
	}

	// write data size
	b := buf.Bytes()
	packetSize := uint32(len(b) - 4)
	binary.BigEndian.PutUint32(b[:4], packetSize)

	// write to connection
	n, err := w.Write(b)

	// return the bytes.Buffer object to bufferPool.
	bufferPool.Put(buf)
	return n, err
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
		buf := bytes.NewBuffer(packetData)
		hp.Unmarshal(buf)
		p = createPacket(hp.PacketType)
		unmarshalPacket(p, buf)
	}); err != nil {
		return nil, packetSize, err
	}
	return p, packetSize, nil
}

func marshalPacket(p xtcp.Packet, buf io.Writer) {
	m := p.(Marshalable)
	m.Marshal(buf)
}

func unmarshalPacket(p xtcp.Packet, r io.Reader) {
	m := p.(Marshalable)
	m.Unmarshal(r)
}

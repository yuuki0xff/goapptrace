package protocol

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/types"
	"github.com/yuuki0xff/xtcp"
)

var (
	rawFuncLogPacket = &RawFuncLogPacket{
		FuncLog: &types.RawFuncLog{
			ID:        types.RawFuncLogID(0x0a0000000000000a),
			Tag:       types.FuncEnd,
			Timestamp: types.Time(0x0b0000000000000b),
			Frames: []uintptr{
				1, 2, 3,
			},
			GID:  types.GID(0x0c0000000000000c),
			TxID: types.TxID(0x0d0000000000000d),
		},
	}
)

// mock of xtcp.Protocol
type fakeProto struct{}

func (fakeProto) PackSize(s xtcp.Packet) int {
	return len(s.String())
}
func (fakeProto) PackTo(p xtcp.Packet, w io.Writer) (int, error) {
	return w.Write([]byte(p.String()))
}
func (fakeProto) PackToByteSlice(p xtcp.Packet, buf []byte) int64 {
	n := copy(buf, []byte(p.String()))
	return int64(n)
}
func (fakeProto) Pack(p xtcp.Packet) ([]byte, error) {
	return []byte(p.String()), nil
}
func (fakeProto) Unpack(b []byte) (xtcp.Packet, int, error) {
	return fakePacket{string(b)}, len(b), nil
}

func fakeMergePacket() *MergePacket {
	return &MergePacket{
		Proto:      &fakeProto{},
		BufferSize: DefaultMaxSmallPacketSize,
	}
}

func BenchmarkDetectPacketType(b *testing.B) {
	p1 := &ClientHelloPacket{}
	p2 := &ServerHelloPacket{}
	p3 := &LogPacket{}
	p4 := &ShutdownPacket{}
	p5 := &StartTraceCmdPacket{}
	p6 := &StopTraceCmdPacket{}
	p7 := &SymbolPacket{}
	p8 := &RawFuncLogPacket{}

	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		detectPacketType(p1)
		detectPacketType(p2)
		detectPacketType(p3)
		detectPacketType(p4)
		detectPacketType(p5)
		detectPacketType(p6)
		detectPacketType(p7)
		detectPacketType(p8)
	}
	b.StopTimer()
}
func BenchmarkCreatePacket(b *testing.B) {
	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		createPacket(ClientHelloPacketType)
		createPacket(ServerHelloPacketType)
		createPacket(LogPacketType)
		createPacket(PingPacketType)
		createPacket(ShutdownPacketType)
		createPacket(StartTraceCmdPacketType)
		createPacket(StopTraceCmdPacketType)
		createPacket(SymbolPacketType)
		createPacket(RawFuncLogPacketType)
	}
	b.StopTimer()
}
func BenchmarkMergePacket_Merge(b *testing.B) {
	mp := MergePacket{
		Proto:      &Proto{},
		BufferSize: DefaultSendBufMaxSize + 2048,
	}
	p := rawFuncLogPacket

	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		mp.Merge(p)
		if mp.Len() >= DefaultSendBufMaxSize {
			mp.Reset()
		}
	}
	b.StopTimer()
}
func BenchmarkRawFuncLogPacket_Marshal(b *testing.B) {
	rp := *rawFuncLogPacket
	buf := make([]byte, DefaultMaxSmallPacketSize)

	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		rp.Marshal(buf)
	}
	b.StopTimer()
}
func BenchmarkRawFuncLogPacket_Unmarshal(b *testing.B) {
	var rp RawFuncLogPacket
	buf := make([]byte, DefaultMaxSmallPacketSize)
	buf = buf[:rawFuncLogPacket.Marshal(buf)]

	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		rp.Unmarshal(buf)
		types.RawFuncLogPool.Put(rp.FuncLog)
	}
	b.StopTimer()
}

func TestPacketType_Marshal(t *testing.T) {
	buf := make([]byte, DefaultMaxSmallPacketSize)
	a := assert.New(t)
	n := PacketType(10).Marshal(buf)
	a.Equal([]byte{10}, buf[:n])
}
func TestPacketType_Unmarshal(t *testing.T) {
	buf := []byte{5}
	a := assert.New(t)
	var pt PacketType
	pt.Unmarshal(buf)
	a.Equal(PacketType(5), pt)
}
func TestMergePacket_Merge(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)

	mp := fakeMergePacket()
	mp.Merge(fakePacket{"<packet 1>"})
	mp.Merge(fakePacket{"<packet 2>"})

	_, err := mp.WriteTo(&buf)
	a.NoError(err)
	a.Equal([]byte("<packet 1><packet 2>"), buf.Bytes())
}
func TestMergePacket_Reset(t *testing.T) {
	a := assert.New(t)

	mp := fakeMergePacket()
	mp.Merge(fakePacket{"abc"})
	a.NotEqual(0, mp.Len())

	mp.Reset()
	a.Equal(0, mp.Len())
}
func TestMergePacket_Len(t *testing.T) {
	a := assert.New(t)

	mp := fakeMergePacket()
	a.Equal(0, mp.Len())

	mp.Merge(fakePacket{"abcd"})
	a.Equal(4, mp.Len())
}
func TestMergePacket_WriteTo(t *testing.T) {
	a := assert.New(t)

	mp := fakeMergePacket()
	mp.Merge(fakePacket{"abcd"})
	a.Equal(4, mp.Len())

	var buf bytes.Buffer
	n, err := mp.WriteTo(&buf)
	a.NoError(err)
	a.Equal(int64(4), n)

	a.Equal(0, mp.Len())
}

package protocol

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/util"
	"github.com/yuuki0xff/xtcp"
)

// mock of xtcp.Protocol
type fakeProto struct{}

func (fakeProto) PackSize(s xtcp.Packet) int {
	return len(s.String())
}
func (fakeProto) PackTo(p xtcp.Packet, w io.Writer) (int, error) {
	return w.Write([]byte(p.String()))
}
func (fakeProto) Pack(p xtcp.Packet) ([]byte, error) {
	return []byte(p.String()), nil
}
func (fakeProto) Unpack(b []byte) (xtcp.Packet, int, error) {
	return fakePacket{string(b)}, len(b), nil
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
		Proto: &Proto{},
	}
	p := &RawFuncLogPacket{
		FuncLog: rawFuncLog,
	}

	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		mp.Merge(p)
		if mp.Len() >= DefaultSendBufferSize {
			mp.Reset()
		}
	}
	b.StopTimer()
}
func BenchmarkRawFuncLogPacket_Marshal(b *testing.B) {
	rp := RawFuncLogPacket{
		FuncLog: rawFuncLog,
	}
	var buf bytes.Buffer
	buf.Grow(1024)

	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		rp.Marshal(&buf)
		buf.Reset()
	}
	b.StopTimer()
}
func BenchmarkRawFuncLogPacket_Unmarshal(b *testing.B) {
	var rp RawFuncLogPacket
	r := util.FakeReader{
		B: rawFuncLogBytes,
	}

	b.ResetTimer()
	for i := b.N; i > 0; i-- {
		rp.Unmarshal(&r)
		// reset fakeReader
		r.N = 0
	}
	b.StopTimer()
}

func TestPacketType_Marshal(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)
	PacketType(10).Marshal(&buf)
	a.Equal([]byte{0, 0, 0, 0, 0, 0, 0, 10}, buf.Bytes())
}
func TestPacketType_Unmarshal(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)
	buf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 5})
	var pt PacketType
	pt.Unmarshal(&buf)
	a.Equal(PacketType(5), pt)
}
func TestMergePacket_Merge(t *testing.T) {
	var buf bytes.Buffer
	a := assert.New(t)

	mp := MergePacket{
		Proto: &fakeProto{},
	}
	mp.Merge(fakePacket{"<packet 1>"})
	mp.Merge(fakePacket{"<packet 2>"})

	_, err := mp.WriteTo(&buf)
	a.NoError(err)
	a.Equal([]byte("<packet 1><packet 2>"), buf.Bytes())
}
func TestMergePacket_Reset(t *testing.T) {
	a := assert.New(t)

	mp := MergePacket{
		Proto: &fakeProto{},
	}
	mp.Merge(fakePacket{"abc"})
	a.NotEqual(0, mp.Len())

	mp.Reset()
	a.Equal(0, mp.Len())
}
func TestMergePacket_Len(t *testing.T) {
	a := assert.New(t)

	mp := MergePacket{
		Proto: &fakeProto{},
	}
	a.Equal(0, mp.Len())

	mp.Merge(fakePacket{"abcd"})
	a.Equal(4, mp.Len())
}
func TestMergePacket_WriteTo(t *testing.T) {
	a := assert.New(t)

	mp := MergePacket{
		Proto: &fakeProto{},
	}
	mp.Merge(fakePacket{"abcd"})
	a.Equal(4, mp.Len())

	var buf bytes.Buffer
	n, err := mp.WriteTo(&buf)
	a.NoError(err)
	a.Equal(int64(4), n)

	a.Equal(0, mp.Len())
}

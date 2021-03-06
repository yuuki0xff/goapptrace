package protocol

import (
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/yuuki0xff/goapptrace/tracer/encoding"
	"github.com/yuuki0xff/goapptrace/tracer/types"
	"github.com/yuuki0xff/xtcp"
)

type PacketType uint8

const (
	fakePacketType = PacketType(iota)
	ClientHelloPacketType
	ServerHelloPacketType
	LogPacketType
	PingPacketType
	ShutdownPacketType
	StartTraceCmdPacketType
	StopTraceCmdPacketType
	SymbolPacketType
	RawFuncLogPacketType
)

// detectPacketType returns PacketType of packet.
// packet MUST be pointer type.
// If packet is not PacketType, will be occurs panic.
func detectPacketType(packet xtcp.Packet) PacketType {
	switch packet.(type) {
	case *fakePacket:
		return fakePacketType
	case *ClientHelloPacket:
		return ClientHelloPacketType
	case *ServerHelloPacket:
		return ServerHelloPacketType
	case *LogPacket:
		return LogPacketType
	case *PingPacket:
		return PingPacketType
	case *ShutdownPacket:
		return ShutdownPacketType
	case *StartTraceCmdPacket:
		return StartTraceCmdPacketType
	case *StopTraceCmdPacket:
		return StopTraceCmdPacketType
	case *SymbolPacket:
		return SymbolPacketType
	case *RawFuncLogPacket:
		return RawFuncLogPacketType
	default:
		log.Panicf("unknown packet type: type=%T value=%+v", packet, packet)
		panic(nil)
	}
}

// createPacket returns empty packet.
func createPacket(packetType PacketType) xtcp.Packet {
	switch packetType {
	case fakePacketType:
		return &fakePacket{}
	case ClientHelloPacketType:
		return &ClientHelloPacket{}
	case ServerHelloPacketType:
		return &ServerHelloPacket{}
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
	case SymbolPacketType:
		return &SymbolPacket{}
	case RawFuncLogPacketType:
		return &RawFuncLogPacket{}
	default:
		log.Panicf("unknown packet type: PacketType=%+v", packetType)
		panic(nil)
	}
}

type Marshalable interface {
	Marshal(buf []byte) int64
	Unmarshal(buf []byte) int64
}
type SizePredictable interface {
	// PacketSize returns encoded byte size.
	PacketSize() int64
}
type DirectWritable interface {
	WriteTo(w io.Writer) (int64, error)
}

func (p PacketType) Marshal(buf []byte) int64 {
	return encoding.MarshalUint8(buf, uint8(p))
}
func (p *PacketType) Unmarshal(buf []byte) int64 {
	val, n := encoding.UnmarshalUint8(buf)
	*p = PacketType(val)
	return n
}

////////////////////////////////////////////////////////////////
// SpecialPacket

// MergePacket can merge several short packets.
// It helps to increase performance by reduce short packets.
type MergePacket struct {
	Proto      ProtoInterface
	BufferSize int
	buff       []byte
	size       int64
}

func (p *MergePacket) String() string { return "<MergePacket>" }
func (p *MergePacket) Merge(packet xtcp.Packet) {
	if p.buff == nil {
		p.buff = make([]byte, p.BufferSize)
	}
	p.size += p.Proto.PackToByteSlice(packet, p.buff[p.size:])
}
func (p *MergePacket) Reset() {
	p.size = 0
}
func (p *MergePacket) Len() int {
	return int(p.size)
}
func (p *MergePacket) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(p.buff[:p.size])
	if err == nil {
		p.size = 0
	}
	return int64(n), err
}

// 巨大なパケットをMergePacketにラップして返す。
// 巨大なパケットを通常の方法でmp.Merge()すると範囲外アクセスでパニックする。この問題を解消するために使用する。
// pは直ぐにmarshalされるため、この関数の実行終了後にpを再利用することが出来る。
func marshalLargePacket(proto ProtoInterface, p xtcp.Packet) *MergePacket {
	mp := &MergePacket{
		Proto: proto,
		// パケットをエンコードするとヘッダーが追加される。
		// ヘッダーのサイズ分だけバッファを大きくする。
		BufferSize: int(p.(SizePredictable).PacketSize()) + PacketHeaderSize,
	}
	mp.Merge(p)
	return mp
}

// fakePacket is a mock of xtcp.Packet.
type fakePacket struct {
	s string
}

func (p fakePacket) String() string {
	return p.s
}
func (p *fakePacket) Marshal(buf []byte) int64 {
	binstr := []byte(p.s)
	return int64(copy(buf, binstr))
}
func (p *fakePacket) Unmarshal(buf []byte) int64 {
	p.s = string(buf)
	return int64(len(buf))
}

////////////////////////////////////////////////////////////////
// HelloPacket

type ClientHelloPacket struct {
	// Process ID
	// 本来はint64にするべきだと思うが、encoderがuint64にしか対応していないため、uint64にした。
	PID uint64
	// Application Name
	AppName string
	// トレーサが動いているHost name
	Host            string
	ClientSecret    string
	ProtocolVersion string
}

type ServerHelloPacket struct {
	ProtocolVersion string
}

func (p ClientHelloPacket) String() string { return "<ClientHelloPacket>" }
func (p ServerHelloPacket) String() string { return "<ServerHelloPacket>" }

func (p *ClientHelloPacket) Marshal(buf []byte) int64 {
	total := encoding.MarshalUint64(buf, p.PID)
	total += encoding.MarshalString(buf[total:], p.AppName)
	total += encoding.MarshalString(buf[total:], p.Host)
	total += encoding.MarshalString(buf[total:], p.ClientSecret)
	total += encoding.MarshalString(buf[total:], p.ProtocolVersion)
	return total
}
func (p *ClientHelloPacket) Unmarshal(buf []byte) int64 {
	var total int64
	var n int64

	p.PID, n = encoding.UnmarshalUint64(buf)
	total += n
	p.AppName, n = encoding.UnmarshalString(buf[total:])
	total += n
	p.Host, n = encoding.UnmarshalString(buf[total:])
	total += n
	p.ClientSecret, n = encoding.UnmarshalString(buf[total:])
	total += n
	p.ProtocolVersion, n = encoding.UnmarshalString(buf[total:])
	total += n
	return total
}
func (p *ServerHelloPacket) Marshal(buf []byte) int64 {
	return encoding.MarshalString(buf, p.ProtocolVersion)
}
func (p *ServerHelloPacket) Unmarshal(buf []byte) int64 {
	var n int64
	p.ProtocolVersion, n = encoding.UnmarshalString(buf)
	return n
}

////////////////////////////////////////////////////////////////
// HeaderPacket

type HeaderPacket struct {
	PacketType PacketType
}

func (p HeaderPacket) String() string {
	return fmt.Sprintf("<HeaderPacket PacketType=%d>",
		p.PacketType)
}
func (p *HeaderPacket) Marshal(buf []byte) int64 {
	return p.PacketType.Marshal(buf)
}
func (p *HeaderPacket) Unmarshal(buf []byte) int64 {
	return p.PacketType.Unmarshal(buf)
}

////////////////////////////////////////////////////////////////
// DataPacket

type LogPacket struct{}
type PingPacket struct{}
type ShutdownPacket struct{}

type StartTraceCmdPacket struct {
	FuncName string
}

type StopTraceCmdPacket struct {
	FuncName string
}

type SymbolPacket struct {
	types.SymbolsData
}
type RawFuncLogPacket struct {
	FuncLog *types.RawFuncLog
}

func (p LogPacket) String() string           { return "<LogPacket>" }
func (p PingPacket) String() string          { return "<PingPacket>" }
func (p ShutdownPacket) String() string      { return "<ShutdownPacket>" }
func (p StartTraceCmdPacket) String() string { return "<StartTraceCmdPacket>" }
func (p StopTraceCmdPacket) String() string  { return "<StopTraceCmdPacket>" }
func (p SymbolPacket) String() string        { return "<SymbolPacket>" }
func (p RawFuncLogPacket) String() string    { return "<RawFuncLogPacket>" }

func (p *LogPacket) Marshal(buf []byte) int64 {
	panic("not implemented")
}
func (p *LogPacket) Unmarshal(buf []byte) int64 {
	panic("not implemented")
}

func (p *PingPacket) Marshal(buf []byte) int64   { return 0 }
func (p *PingPacket) Unmarshal(buf []byte) int64 { return 0 }

func (p *ShutdownPacket) Marshal(buf []byte) int64   { return 0 }
func (p *ShutdownPacket) Unmarshal(buf []byte) int64 { return 0 }

func (p *StartTraceCmdPacket) Marshal(buf []byte) int64   { return slowMarshal(buf, p) }
func (p *StartTraceCmdPacket) Unmarshal(buf []byte) int64 { return slowUnmarshal(buf, p) }

func (p *StopTraceCmdPacket) Marshal(buf []byte) int64   { return slowMarshal(buf, p) }
func (p *StopTraceCmdPacket) Unmarshal(buf []byte) int64 { return slowUnmarshal(buf, p) }

func (p *SymbolPacket) Marshal(buf []byte) int64 {
	return encoding.MarshalSymbolsData(&p.SymbolsData, buf)
}
func (p *SymbolPacket) Unmarshal(buf []byte) int64 {
	return encoding.UnmarshalSymbolsData(&p.SymbolsData, buf)
}
func (p *SymbolPacket) PacketSize() int64 {
	return encoding.SizeSymbolsData(&p.SymbolsData)
}

func (p *RawFuncLogPacket) Marshal(buf []byte) int64 {
	return encoding.MarshalRawFuncLog(buf, p.FuncLog)
}
func (p *RawFuncLogPacket) Unmarshal(buf []byte) int64 {
	var n int64
	fl := types.RawFuncLogPool.Get().(*types.RawFuncLog)
	fl.Frames = fl.Frames[:cap(fl.Frames)]
	n = encoding.UnmarshalRawFuncLog(buf, fl)
	p.FuncLog = fl
	return n
}

// slowMarshal encodes v and save into buf.
func slowMarshal(buf []byte, v interface{}) int64 {
	js, err := json.Marshal(v)
	if err != nil {
		log.Panicln(err)
	}
	total := encoding.MarshalUint64(buf, uint64(len(js)))
	total += int64(copy(buf[total:], js))
	return total
}

// slowUnmarshal decodes v from buf.
func slowUnmarshal(buf []byte, v interface{}) int64 {
	var total int64
	length, n := encoding.UnmarshalUint64(buf)
	total += n
	err := json.Unmarshal(buf[total:total+int64(length)], v)
	if err != nil {
		log.Panicln(err)
	}
	total += int64(length)
	return total
}

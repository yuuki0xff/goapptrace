package protocol

import (
	"fmt"
	"io"
	"log"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/xtcp"
)

type PacketType uint64

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
	return marshalUint64(buf, uint64(p))
}
func (p *PacketType) Unmarshal(buf []byte) int64 {
	val, n := unmarshalUint64(buf)
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

// ByteWrapPacket wraps []byte.
// Main purpose is to send large packets.
type ByteWrapPacket struct {
	Buff []byte
}

func (p *ByteWrapPacket) String() string { return "<ByteWrapPacket>" }
func (p *ByteWrapPacket) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(p.Buff)
	return int64(n), err
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
	AppName         string
	ClientSecret    string
	ProtocolVersion string
}

type ServerHelloPacket struct {
	ProtocolVersion string
}

func (p ClientHelloPacket) String() string { return "<ClientHelloPacket>" }
func (p ServerHelloPacket) String() string { return "<ServerHelloPacket>" }

func (p *ClientHelloPacket) Marshal(buf []byte) int64 {
	total := marshalString(buf, p.AppName)
	total += marshalString(buf[total:], p.ClientSecret)
	total += marshalString(buf[total:], p.ProtocolVersion)
	return total
}
func (p *ClientHelloPacket) Unmarshal(buf []byte) int64 {
	var total int64
	var n int64

	p.AppName, n = unmarshalString(buf)
	total += n
	p.ClientSecret, n = unmarshalString(buf[total:])
	total += n
	p.ProtocolVersion, n = unmarshalString(buf[total:])
	total += n
	return total
}
func (p *ServerHelloPacket) Marshal(buf []byte) int64 {
	return marshalString(buf, p.ProtocolVersion)
}
func (p *ServerHelloPacket) Unmarshal(buf []byte) int64 {
	var n int64
	p.ProtocolVersion, n = unmarshalString(buf)
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
	FuncEntry  uintptr
	ModuleName string
}
type StopTraceCmdPacket struct {
	FuncEntry  uintptr
	ModuleName string
}

type SymbolPacket struct {
	logutil.SymbolsData
}
type RawFuncLogPacket struct {
	FuncLog *logutil.RawFuncLog
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

func (p *StartTraceCmdPacket) Marshal(buf []byte) int64 {
	total := marshalUintptr(buf, p.FuncEntry)
	total += marshalString(buf[total:], p.ModuleName)
	return total
}
func (p *StartTraceCmdPacket) Unmarshal(buf []byte) int64 {
	var total int64
	var n int64
	p.FuncEntry, n = unmarshalUintptr(buf)
	total += n
	p.ModuleName, n = unmarshalString(buf[total:])
	total += n
	return total
}

func (p *StopTraceCmdPacket) Marshal(buf []byte) int64 {
	total := marshalUintptr(buf, p.FuncEntry)
	total += marshalString(buf[total:], p.ModuleName)
	return total
}
func (p *StopTraceCmdPacket) Unmarshal(buf []byte) int64 {
	var total int64
	var n int64
	p.FuncEntry, n = unmarshalUintptr(buf)
	total += n
	p.ModuleName, n = unmarshalString(buf)
	total += n
	return total
}

func (p *SymbolPacket) Marshal(buf []byte) int64 {
	// TODO: marshal Files
	total := marshalStringSlice(buf, p.Files)
	// TODO: marshal Mods
	total += marshalGoModuleSlice(buf[total:], p.Mods)
	total += marshalGoFuncSlice(buf[total:], p.Funcs)
	total += marshalGoLineSlice(buf[total:], p.Lines)
	return total
}
func (p *SymbolPacket) Unmarshal(buf []byte) int64 {
	var total int64
	var n int64
	// TODO: unmarshal Files
	p.Files, n = unmarshalStringSlice(buf)
	total += n
	// TODO: unmarshal Mods
	p.Mods, n = unmarshalGoModuleSlice(buf[total:])
	total += n
	p.Funcs, n = unmarshalGoFuncSlice(buf[total:])
	total += n
	p.Lines, n = unmarshalGoLineSlice(buf[total:])
	total += n
	return total
}

func (p *RawFuncLogPacket) Marshal(buf []byte) int64 {
	return marshalRawFuncLog(buf, p.FuncLog)
}
func (p *RawFuncLogPacket) Unmarshal(buf []byte) int64 {
	var n int64
	p.FuncLog, n = unmarshalRawFuncLog(buf)
	return n
}

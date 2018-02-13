package protocol

import (
	"bytes"
	"fmt"
	"io"
	"log"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/xtcp"
)

type PacketType uint64

const (
	UnknownPacketType = PacketType(iota)
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
	Marshal(w io.Writer)
	Unmarshal(r io.Reader)
}

func (p PacketType) Marshal(w io.Writer) {
	marshalUint64(w, uint64(p))
}
func (p *PacketType) Unmarshal(r io.Reader) {
	val := unmarshalUint64(r)
	*p = PacketType(val)
}

////////////////////////////////////////////////////////////////
// SpecialPacket

// MergePacket can merge several short packets.
// It helps to increase performance by reduce short packets.
type MergePacket struct {
	Proto xtcp.Protocol
	buff  bytes.Buffer
}

func (p *MergePacket) String() string { return "<MergePacket>" }
func (p *MergePacket) Merge(packet xtcp.Packet) {
	_, err := p.Proto.PackTo(packet, &p.buff)
	if err != nil {
		log.Panic(err)
	}
}
func (p *MergePacket) Reset() {
	p.buff.Reset()
}
func (p *MergePacket) Len() int {
	return p.buff.Len()
}
func (p *MergePacket) WriteTo(w io.Writer) (int64, error) {
	return p.buff.WriteTo(w)
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

func (p *ClientHelloPacket) Marshal(w io.Writer) {
	marshalString(w, p.AppName)
	marshalString(w, p.ClientSecret)
	marshalString(w, p.ProtocolVersion)
}
func (p *ClientHelloPacket) Unmarshal(r io.Reader) {
	p.AppName = unmarshalString(r)
	p.ClientSecret = unmarshalString(r)
	p.ProtocolVersion = unmarshalString(r)
}
func (p *ServerHelloPacket) Marshal(w io.Writer) {
	marshalString(w, p.ProtocolVersion)
}
func (p *ServerHelloPacket) Unmarshal(r io.Reader) {
	p.ProtocolVersion = unmarshalString(r)
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
func (p *HeaderPacket) Marshal(w io.Writer) {
	p.PacketType.Marshal(w)
}
func (p *HeaderPacket) Unmarshal(r io.Reader) {
	p.PacketType.Unmarshal(r)
}

////////////////////////////////////////////////////////////////
// DataPacket

type LogPacket struct{}
type PingPacket struct{}
type ShutdownPacket struct{}

type StartTraceCmdPacket struct {
	FuncID     logutil.FuncID
	ModuleName string
}
type StopTraceCmdPacket struct {
	FuncID     logutil.FuncID
	ModuleName string
}

type SymbolPacket struct {
	logutil.SymbolsDiff
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

func (p *LogPacket) Marshal(w io.Writer) {
	panic("not implemented")
}
func (p *LogPacket) Unmarshal(r io.Reader) {
	panic("not implemented")
}

func (p *PingPacket) Marshal(w io.Writer)   {}
func (p *PingPacket) Unmarshal(r io.Reader) {}

func (p *ShutdownPacket) Marshal(w io.Writer)   {}
func (p *ShutdownPacket) Unmarshal(r io.Reader) {}

func (p *StartTraceCmdPacket) Marshal(w io.Writer) {
	marshalFuncID(w, p.FuncID)
	marshalString(w, p.ModuleName)
}
func (p *StartTraceCmdPacket) Unmarshal(r io.Reader) {
	p.FuncID = unmarshalFuncID(r)
	p.ModuleName = unmarshalString(r)
}

func (p *StopTraceCmdPacket) Marshal(w io.Writer) {
	marshalFuncID(w, p.FuncID)
	marshalString(w, p.ModuleName)
}
func (p *StopTraceCmdPacket) Unmarshal(r io.Reader) {
	p.FuncID = unmarshalFuncID(r)
	p.ModuleName = unmarshalString(r)
}

func (p *SymbolPacket) Marshal(w io.Writer) {
	marshalFuncSymbolSlice(w, p.Funcs)
	marshalFuncStatusSlice(w, p.FuncStatus)
}
func (p *SymbolPacket) Unmarshal(r io.Reader) {
	p.Funcs = unmarshalFuncSymbolSlice(r)
	p.FuncStatus = unmarshalFuncStatusSlice(r)
}

func (p *RawFuncLogPacket) Marshal(w io.Writer) {
	marshalRawFuncLog(w, p.FuncLog)
}
func (p *RawFuncLogPacket) Unmarshal(r io.Reader) {
	p.FuncLog = unmarshalRawFuncLog(r)
}

package protocol

import (
	"fmt"
	"io"
	"reflect"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	. "github.com/yuuki0xff/goapptrace/tracer/util"
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
	RawFuncLogNewPacketType
)

// detectPacketType returns PacketType of packet.
// If packet is not PacketType, will be occurs panic.
func detectPacketType(packet xtcp.Packet) PacketType {
	if reflect.TypeOf(packet).Kind() == reflect.Ptr {
		// dereference the packet of pointer type.
		packet = reflect.ValueOf(packet).Elem().Interface().(xtcp.Packet)
	}

	switch packet.(type) {
	case ClientHelloPacket:
		return ClientHelloPacketType
	case ServerHelloPacket:
		return ServerHelloPacketType
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
	case SymbolPacket:
		return SymbolPacketType
	case RawFuncLogNewPacket:
		return RawFuncLogNewPacketType
	default:
		return UnknownPacketType
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
	case RawFuncLogNewPacketType:
		return &RawFuncLogNewPacket{}
	default:
		return nil
	}
}

type Marshalable interface {
	Marshal(w io.Writer) error
	Unmarshal(r io.Reader) error
}

func (p PacketType) Marshal(w io.Writer) error {
	return PanicHandler(func() {
		marshalUint64(w, uint64(p))
	})
}
func (p *PacketType) Unmarshal(r io.Reader) error {
	return PanicHandler(func() {
		val := unmarshalUint64(r)
		*p = PacketType(val)
	})
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

func (p *ClientHelloPacket) Marshal(w io.Writer) error {
	return PanicHandler(func() {
		marshalString(w, p.AppName)
		marshalString(w, p.ClientSecret)
		marshalString(w, p.ProtocolVersion)
	})
}
func (p *ClientHelloPacket) Unmarshal(r io.Reader) error {
	return PanicHandler(func() {
		p.AppName = unmarshalString(r)
		p.ClientSecret = unmarshalString(r)
		p.ProtocolVersion = unmarshalString(r)
	})
}
func (p *ServerHelloPacket) Marshal(w io.Writer) error {
	return PanicHandler(func() {
		marshalString(w, p.ProtocolVersion)
	})
}
func (p *ServerHelloPacket) Unmarshal(r io.Reader) error {
	return PanicHandler(func() {
		p.ProtocolVersion = unmarshalString(r)
	})
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
func (p *HeaderPacket) Marshal(w io.Writer) error {
	return p.PacketType.Marshal(w)
}
func (p *HeaderPacket) Unmarshal(r io.Reader) error {
	return p.PacketType.Unmarshal(r)
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
type RawFuncLogNewPacket struct {
	FuncLog *logutil.RawFuncLog
}

func (p LogPacket) String() string           { return "<LogPacket>" }
func (p PingPacket) String() string          { return "<PingPacket>" }
func (p ShutdownPacket) String() string      { return "<ShutdownPacket>" }
func (p StartTraceCmdPacket) String() string { return "<StartTraceCmdPacket>" }
func (p StopTraceCmdPacket) String() string  { return "<StopTraceCmdPacket>" }
func (p SymbolPacket) String() string        { return "<SymbolPacket>" }
func (p RawFuncLogNewPacket) String() string { return "<RawFuncLogNewPacket>" }

func (p *LogPacket) Marshal(w io.Writer) error {
	panic("not implemented")
}
func (p *LogPacket) Unmarshal(r io.Reader) error {
	panic("not implemented")
}

func (p *PingPacket) Marshal(w io.Writer) error   { return nil }
func (p *PingPacket) Unmarshal(r io.Reader) error { return nil }

func (p *ShutdownPacket) Marshal(w io.Writer) error   { return nil }
func (p *ShutdownPacket) Unmarshal(r io.Reader) error { return nil }

func (p *StartTraceCmdPacket) Marshal(w io.Writer) error {
	return PanicHandler(func() {
		marshalFuncID(w, p.FuncID)
		marshalString(w, p.ModuleName)
	})
}
func (p *StartTraceCmdPacket) Unmarshal(r io.Reader) error {
	return PanicHandler(func() {
		p.FuncID = unmarshalFuncID(r)
		p.ModuleName = unmarshalString(r)
	})
}

func (p *StopTraceCmdPacket) Marshal(w io.Writer) error {
	return PanicHandler(func() {
		marshalFuncID(w, p.FuncID)
		marshalString(w, p.ModuleName)
	})
}
func (p *StopTraceCmdPacket) Unmarshal(r io.Reader) error {
	return PanicHandler(func() {
		p.FuncID = unmarshalFuncID(r)
		p.ModuleName = unmarshalString(r)
	})
}

func (p *SymbolPacket) Marshal(w io.Writer) error {
	return PanicHandler(func() {
		marshalFuncSymbolSlice(w, p.Funcs)
		marshalFuncStatusSlice(w, p.FuncStatus)
	})
}
func (p *SymbolPacket) Unmarshal(r io.Reader) error {
	return PanicHandler(func() {
		p.Funcs = unmarshalFuncSymbolSlice(r)
		p.FuncStatus = unmarshalFuncStatusSlice(r)
	})
}

func (p *RawFuncLogNewPacket) Marshal(w io.Writer) error {
	return PanicHandler(func() {
		marshalRawFuncLog(w, p.FuncLog)
	})
}
func (p *RawFuncLogNewPacket) Unmarshal(r io.Reader) error {
	return PanicHandler(func() {
		p.FuncLog = unmarshalRawFuncLog(r)
	})
}

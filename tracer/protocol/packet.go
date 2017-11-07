package protocol

import (
	"fmt"
	"log"

	"reflect"

	"github.com/xfxdev/xtcp"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

type PacketType int64

const (
	UnknownPacketType = PacketType(iota)
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
		log.Panic(fmt.Sprintf("bug: invalid Packet: %+v", packet))
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
	case SymbolPacketType:
		return &SymbolPacket{}
	case RawFuncLogNewPacketType:
		return &RawFuncLogNewPacket{}
	default:
		log.Panic(fmt.Sprintf("bug: invalid PacketType: %+v", packetType))
		return nil
	}
}

type HeaderPacket struct {
	PacketType PacketType
}

type LogPacket struct{}
type PingPacket struct{}
type ShutdownPacket struct{}

type StartTraceCmdPacket struct{}
type StopTraceCmdPacket struct{}

type SymbolPacket struct {
	Symbols *logutil.Symbols
}
type RawFuncLogNewPacket struct {
	FuncLog *logutil.RawFuncLogNew
}

func (p HeaderPacket) String() string {
	return fmt.Sprintf("<HeaderPacket PacketType=%d>",
		p.PacketType)
}
func (p ClientHelloPacket) String() string   { return "<ClientHelloPacket>" }
func (p ServerHelloPacket) String() string   { return "<ServerHelloPacket>" }
func (p LogPacket) String() string           { return "<LogPacket>" }
func (p PingPacket) String() string          { return "<PingPacket>" }
func (p ShutdownPacket) String() string      { return "<ShutdownPacket>" }
func (p StartTraceCmdPacket) String() string { return "<StartTraceCmdPacket>" }
func (p StopTraceCmdPacket) String() string  { return "<StopTraceCmdPacket>" }
func (p MessageHeader) String() string       { return "<MessageHeader>" }
func (p SymbolPacket) String() string        { return "<SymbolPacket>" }
func (p RawFuncLogNewPacket) String() string { return "<RawFuncLogNewPacket>" }

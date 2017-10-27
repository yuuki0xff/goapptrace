package protocol

import (
	"fmt"
	"log"

	"github.com/xfxdev/xtcp"
)

type PacketType int64

const (
	UnknownPacketType = PacketType(iota)
	LogPacketType
	PingPacketType
	ShutdownPacketType
	StartTraceCmdPacketType
	StopTraceCmdPacketType
)

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

type HeaderPacket struct {
	PacketType PacketType
}

type LogPacket struct{}
type PingPacket struct{}
type ShutdownPacket struct{}

type StartTraceCmdPacket struct{}
type StopTraceCmdPacket struct{}

func (p HeaderPacket) String() string {
	return fmt.Sprintf("<HeaderPacket PacketType=%d>",
		p.PacketType)
}
func (p LogPacket) String() string {
	return ""
}
func (p PingPacket) String() string {
	return ""
}
func (p ShutdownPacket) String() string {
	return ""
}
func (p StartTraceCmdPacket) String() string {
	return ""
}
func (p StopTraceCmdPacket) String() string {
	return ""
}

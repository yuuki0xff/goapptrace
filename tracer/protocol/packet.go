package protocol

import "fmt"

type PacketType int64

const (
	UnknownPacketType = PacketType(iota)
	LogPacketType
	PingPacketType
	ShutdownPacketType
	StartTraceCmdPacketType
	StopTraceCmdPacketType
)

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

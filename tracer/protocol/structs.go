package protocol

type MessageType uint64
type FuncLogType uint64
type CommandType uint64

const (
	FuncStart FuncLogType = iota
	FuncEnd
)

////////////////////////////////////////////////////////////////
// Headers

type ClientHelloPacket struct {
	AppName         string
	ClientSecret    string
	ProtocolVersion string
}

type ServerHelloPacket struct {
	ProtocolVersion string
}

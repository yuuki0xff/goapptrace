package protocol

import "github.com/yuuki0xff/goapptrace/tracer/log"

// Protocol Specification
// Step1: Server <-- Client
//   [ClientHeader]
//
// Step2: Server --> Client
//   [ServerHeader]
//
// Step3: Server <-> Client
//   Client -> Server
//     [MessageHeader] [MessageData]*n
//     [MessageHeader] [MessageData]*n
//     ...
//   Server -> Client
//     [CommandHeader] [CommandArgs]
//     [CommandHeader] [CommandArgs]
//     ...

type MessageType uint64
type FuncLogType uint64
type CommandType uint64

const (
	PingMsg MessageType = iota
	SymbolsMsg
	RawFuncLogMsg
)

const (
	FuncStart FuncLogType = iota
	FuncEnd
)

const (
	PingCmd CommandType = iota
	StartTraceCmd
	StopTraceCmd
)

////////////////////////////////////////////////////////////////
// Headers

type ClientHeader struct {
	AppName       string
	ClientSecret  string
	ClientVersion string
}

type MessageHeader struct {
	MessageType MessageType
	Messages    uint64 // number of messages
}

type ServerHeader struct {
	ServerVersion string
}

type CommandHeader struct {
	CommandType CommandType
}

////////////////////////////////////////////////////////////////
// Messages

type PingMsgData struct {
}

////////////////////////////////////////////////////////////////
// Command Arguments

type PingCmdArgs struct {
}

type StartTraceCmdArgs struct {
	FuncID log.FuncID
}

type StopTraceCmdArgs struct {
	FuncID log.FuncID
}

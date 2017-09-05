package protocol

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
type FuncID uint64
type FuncStatusID uint64
type TxID uint64
type FuncLogType uint64

type CommandType uint64

const (
	PingMsg MessageType = iota
	SymbolsMsg
	FuncLogMsg
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

type Symbols struct {
	Funcs          []*Func
	FuncExecStatus []*FuncStatus
}

type Func struct {
	ID   FuncID
	Name string
	File string
}

type FuncStatus struct {
	ID    FuncStatusID
	Func  FuncID
	Line  uint64
	Entry uintptr
}

type FuncLog struct {
	TxID      TxID
	Timestamp int64
	Frames    []FuncStatusID
}

////////////////////////////////////////////////////////////////
// Command Arguments

type StartTraceCmdArgs struct {
	FuncID FuncID
}

type StopTraceCmdArgs struct {
	FuncID FuncID
}

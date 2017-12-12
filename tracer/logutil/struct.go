package logutil

import "runtime"

const (
	NotEnded      = -1
	TimeRangeStep = 5000

	FuncStart = TagName("funcStart")
	FuncEnd   = TagName("funcEnd")
)

type LoadRawLogHandler func(*RawFuncLogNew)
type LoadFuncLogHandler func(*FuncLog)
type GID int64 // GID - Goroutine ID
type TxID uint64
type Time int
type TagName string
type TimeRange struct{ rangeID int }
type RecordList []*FuncLog
type GoroutineMap struct {
	m map[GID]*Goroutine
}
type TimeRangeMap struct {
	m map[TimeRange]*GoroutineMap
}

// TODO: 要リファクタ
type RawLogLoader struct {
	Name string

	RawLogHandler  LoadRawLogHandler
	FuncLogHandler LoadFuncLogHandler

	////////////////
	// ↓ 初期化不要 ↓
	Symbols       Symbols
	SymbolsEditor SymbolsEditor
	Records       RecordList
	GoroutineMap  *GoroutineMap
	TimeRangeMap  *TimeRangeMap
}

type Goroutine struct {
	GID       GID
	Records   RecordList
	StartTime Time
	EndTime   Time
}

type FuncLog struct {
	StartTime Time
	EndTime   Time
	Parent    *FuncLog

	Frames []FuncStatusID
	GID    GID
}

type RawFuncLog struct {
	Time      Time
	Tag       string          `json:"tag"`
	Timestamp int64           `json:"timestamp"`
	Frames    []runtime.Frame `json:"frames"`
	GID       GID             `json:"gid"`
	TxID      TxID            `json:"txid"`
}

type RawFuncLogNew struct {
	Time      Time
	Tag       TagName        `json:"tag"`
	Timestamp int64          `json:"timestamp"`
	Frames    []FuncStatusID `json:"frames"` // Frames[0] is current frame, Frames[1] is the caller of Frame[0].
	GID       GID            `json:"gid"`
	TxID      TxID           `json:"txid"`
}

////////////////////////////////////////////////////////////////
// Symbols
type FuncID uint64
type FuncStatusID uint64

type Symbols struct {
	Funcs      []*FuncSymbol
	FuncStatus []*FuncStatus
}

type FuncSymbol struct {
	ID    FuncID
	Name  string  // example: "github.com/yuuki0xff/goapptrace.main"
	File  string  // example: "/go/src/github.com/yuuki0xff/goapptrace/goapptrace.go"
	Entry uintptr // entry point of function
}

type FuncStatus struct {
	ID   FuncStatusID
	Func FuncID
	Line uint64
	PC   uintptr
}

type SymbolsEditor struct {
	KeepID     bool
	symbols    *Symbols
	funcs      map[string]FuncID
	funcStatus map[FuncStatus]FuncStatusID // FuncStatus.IDは常に0
}

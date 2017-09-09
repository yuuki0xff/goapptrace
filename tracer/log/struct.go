package log

import "runtime"

const (
	NotEnded      = -1
	TimeRangeStep = 5000
)

type LoadRawLogHandler func(*RawLog)
type LoadFuncLogHandler func(*FuncLog)
type GID int64 // GID - Goroutine ID
type TxID uint64
type Time int
type TimeRange struct{ rangeID int }
type RecordList []*FuncLog
type GoroutineMap struct {
	m map[GID]*Goroutine
}
type TimeRangeMap struct {
	m map[TimeRange]*GoroutineMap
}

type RawLogLoader struct {
	Name         string
	Records      RecordList
	GoroutineMap *GoroutineMap
	TimeRangeMap *TimeRangeMap

	RawLogHandler  LoadRawLogHandler
	FuncLogHandler LoadFuncLogHandler
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

	Frames []runtime.Frame
	GID    GID
}

type RawLog struct {
	Time      Time
	Tag       string          `json:"tag"`
	Timestamp int64           `json:"timestamp"`
	Frames    []runtime.Frame `json:"frames"`
	GID       GID             `json:"gid"`
	TxID      TxID            `json:"txid"`
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

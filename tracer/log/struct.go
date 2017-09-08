package log

import "runtime"

const (
	NotEnded      = -1
	TimeRangeStep = 5000
)

type LoadRawLogHandler func(*RawLog)
type LoadFuncLogHandler func(*FuncLog)
type GID int // GID - Goroutine ID
type Time int
type TimeRange struct{ rangeID int }
type RecordList []*FuncLog
type GoroutineMap struct {
	m map[GID]*Goroutine
}
type TimeRangeMap struct {
	m map[TimeRange]*GoroutineMap
}

type Log struct {
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
	Timestamp int             `json:"timestamp"`
	Frames    []runtime.Frame `json:"frames"`
	GID       GID             `json:"gid"`
}

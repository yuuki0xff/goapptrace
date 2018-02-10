package logviewer

import "github.com/yuuki0xff/goapptrace/tracer/logutil"

// UIState is status of Coordinator.
type UIState struct {
	LogID        string
	RecordID     logutil.FuncLogID
	UseGraphView bool
}

// LLState is status of LogListVM.
type LLState int

// LRState is status of LogRecordVM.
type LRState int

const (
	LLLoadingState LLState = iota
	LLWait
	LLSelectedState

	LRLoadingState LRState = iota
	LRWait
	LRSelectedState
)

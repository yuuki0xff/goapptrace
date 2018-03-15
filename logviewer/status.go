package logviewer

import (
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

// UIState is status of Coordinator.
type UIState struct {
	LogID        string
	RecordID     types.FuncLogID
	Record       types.FuncLog
	UseGraphView bool
}

// LLState is status of LogListVM.
type LLState int

// LRState is status of LogRecordVM.
type LRState int

// FCDState is status of FuncLogDetailVM.
type FCDState int

// GState is status of GraphVM
type GState int

const (
	LLLoadingState LLState = iota
	LLWait
	LLSelectedState
)
const (
	LRLoadingState LRState = iota
	LRWait
	LRSelectedState
)
const (
	FCDLoading FCDState = iota
	FCDWait
)
const (
	GLoading GState = iota
	GWait
)

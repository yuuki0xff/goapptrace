package logviewer

import (
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

// UIState is status of Coordinator.
type UIState struct {
	LogID        string
	RecordID     logutil.FuncLogID
	Record       restapi.FuncCall
	UseGraphView bool
}

// LLState is status of LogListVM.
type LLState int

// LRState is status of LogRecordVM.
type LRState int

// FCDState is status of FuncCallDetailVM.
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

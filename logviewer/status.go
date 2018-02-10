package logviewer

// UIState is status of Coordinator.
type UIState struct {
	LogID        string
	RecordID     string
	UseGraphView bool
}

// LLState is status of LogListVM.
type LLState int

const (
	LLLoadingState LLState = iota
	LLWait
	LLSelectedState
)

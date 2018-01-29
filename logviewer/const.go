package logviewer

import "time"

const (
	LoadingText = "Loading ..."
	NoLogFiles  = "Available logs not found"
	ErrorText   = "Error"

	StatusRunningText = "Running"
	StatusStoppedText = ""
	RunningStyleName  = "status-running"
	StoppedStyleName  = "status-stopped"

	APIConnections = 4
	UpdateInterval = 500 * time.Millisecond
)

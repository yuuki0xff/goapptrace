package logviewer

import "time"

const (
	LoadingText    = "Loading ..."
	NoLogFiles     = "Available logs not found"
	ErrorText      = "Error"
	APIConnections = 4
	UpdateInterval = 500 * time.Millisecond
)

package logviewer

import (
	"github.com/marcusolsson/tui-go"
)

type ErrorMsg struct {
	tui.Widget
	Err error
}

func newErrorMsg(err error) *ErrorMsg {
	label := tui.NewLabel("ERROR: " + err.Error())
	label.SetStyleName("error-message")
	return &ErrorMsg{
		Widget: label,
		Err:    err,
	}
}

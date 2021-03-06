package logviewer

import (
	"github.com/marcusolsson/tui-go"
)

// Coordinator manages the navigation flow.
type Coordinator interface {
	// SetState sets status of the Coordinator to s.
	SetState(s UIState)
	// NotifyVMUpdated notifies that current ViewModel is updated.
	// Coordinator MUST replace views.
	NotifyVMUpdated()
}

// ViewModel interface implements the business logic.
type ViewModel interface {
	Updatable
	// View builds View object from cache.
	View() View
}

// View interface creates widgets to build the user interface,
// and notify user inputs to ViewModel interface.
type View interface {
	// Widget creates widgets, and returns a parent widget.
	Widget() tui.Widget
	// Keybindings returns map of key-sequences and handlers.
	// If we do not necessarily set keybindings, it will return nil or empty map.
	Keybindings() map[string]func()
	// FocusChain returns tui.FocusChain.
	// You MUST call the View.Widget() before call this method.
	FocusChain() tui.FocusChain
}

func newFocusChain(widgets ...tui.Widget) tui.FocusChain {
	fc := &tui.SimpleFocusChain{}
	fc.Set(widgets...)
	return fc
}

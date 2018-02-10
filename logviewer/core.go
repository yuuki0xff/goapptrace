package logviewer

import (
	"github.com/yuuki0xff/tui-go"
)

type State int

// Coordinator manages the navigation flow.
type Coordinator interface {
	HandleEvent(vm ViewModel, s State)
}

// ViewModel interface implements the business logic.
type ViewModel interface {
	Updatable
	// SetState sets state to s.
	SetState(s State)
	// Paint builds tui.Widgets from cache.
	Paint()
	// LastUpdate returns version number of cache.
	// ViewModel MUST increase the version number if caches are updated.
	// View can skip rebuild widgets if Version() is equal to previous version.
	Version() int
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

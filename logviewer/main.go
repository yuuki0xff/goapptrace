package logviewer

import (
	"github.com/marcusolsson/tui-go"
	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

type View interface {
	Widget() tui.Widget
	SetKeybindings()
	Quit()
}

type LogViewer struct {
	Config *config.Config
	Api    *restapi.Client
	LogID  string
	ui     tui.UI
	view   View
}

func (v *LogViewer) Run() error {
	v.view = &selectLogView{
		root: v,
	}

	v.ui = tui.New(v.view.Widget())
	v.setKeybindings()

	if err := v.ui.Run(); err != nil {
		return errors.Wrap(err, "failed to initialize TUI")
	}
	return nil
}
func (v *LogViewer) Quit() {
	v.ui.Quit()
}
func (v *LogViewer) setKeybindings() {
	v.ui.SetKeybinding("Q", v.Quit)
	v.ui.SetKeybinding("Esc", v.Quit)
}
func (v *LogViewer) setView(view View) {
	v.view = view
	v.ui.SetWidget(v.view.Widget())

	// rebuild key bind settings.
	v.ui.ClearKeybindings()
	v.setKeybindings()
	v.view.SetKeybindings()
}

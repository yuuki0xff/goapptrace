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
	UI     tui.UI
	view   View
}

func (v *LogViewer) Run() error {
	var err error
	v.view = &selectLogView{
		root: v,
	}

	v.UI, err = tui.New(v.view.Widget())
	if err != nil {
		return errors.Wrap(err, "failed to initialize TUI")
	}
	v.setKeybindings()

	if err := v.UI.Run(); err != nil {
		return errors.Wrap(err, "failed to initialize TUI")
	}
	return nil
}
func (v *LogViewer) Quit() {
	v.UI.Quit()
}
func (v *LogViewer) setKeybindings() {
	v.UI.SetKeybinding("Q", v.Quit)
	v.UI.SetKeybinding("Esc", v.Quit)
}
func (v *LogViewer) setView(view View) {
	v.view = view
	v.UI.SetWidget(v.view.Widget())

	// rebuild key bind settings.
	v.UI.ClearKeybindings()
	v.setKeybindings()
	v.view.SetKeybindings()
}

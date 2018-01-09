package logviewer

import (
	"github.com/marcusolsson/tui-go"
	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

type View interface {
	Widget() tui.Widget
	// 画面を更新する
	Update()
	SetKeybindings()
	Quit()
}

type Controller struct {
	Config *config.Config
	Api    *restapi.Client
	LogID  string
	UI     tui.UI
	view   View
}

func (v *Controller) Run() error {
	var err error
	v.view = &selectLogView{
		Root: v,
	}

	v.UI, err = tui.New(tui.NewSpacer())
	if err != nil {
		return errors.Wrap(err, "failed to initialize TUI")
	}
	v.setView(v.view)

	if err := v.UI.Run(); err != nil {
		return errors.Wrap(err, "failed to initialize TUI")
	}
	return nil
}
func (v *Controller) Quit() {
	v.UI.Quit()
}
func (v *Controller) setKeybindings() {
	v.UI.SetKeybinding("Q", v.Quit)
	v.UI.SetKeybinding("Esc", v.Quit)
}
func (v *Controller) setView(view View) {
	v.view = view
	v.UI.SetWidget(v.view.Widget())

	// rebuild key bind settings.
	v.UI.ClearKeybindings()
	v.setKeybindings()
	v.view.SetKeybindings()

	go v.UI.Update(v.view.Update)
}

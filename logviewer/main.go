package logviewer

import (
	"github.com/marcusolsson/tui-go"
	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

type View interface {
	tui.Widget
	// 画面を更新する
	Update()
	SetKeybindings()
	FocusChain() tui.FocusChain
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
	v.view = newSelectLogView(v)

	v.UI, err = tui.New(tui.NewSpacer())
	if err != nil {
		return errors.Wrap(err, "failed to initialize TUI")
	}
	theme := tui.NewTheme()
	theme.SetStyle("list.item.selected", tui.Style{Reverse: tui.DecorationOn})
	theme.SetStyle("table.cell.selected", tui.Style{Reverse: tui.DecorationOn})
	theme.SetStyle("button.focused", tui.Style{Reverse: tui.DecorationOn})
	theme.SetStyle("label.error-message", tui.Style{
		Fg:   tui.ColorRed,
		Bold: tui.DecorationOn,
	})
	v.UI.SetTheme(theme)
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
	v.UI.SetWidget(v.view)

	// rebuild key bind settings.
	v.UI.ClearKeybindings()
	v.setKeybindings()
	v.view.SetKeybindings()

	// update focus chain
	v.UI.SetFocusChain(v.view.FocusChain())

	go v.UI.Update(v.view.Update)
}

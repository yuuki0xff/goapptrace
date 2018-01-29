package logviewer

import (
	"context"

	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/tui-go"
)

type View interface {
	tui.Widget
	// 画面を更新する
	Update()
	SetKeybindings()
	FocusChain() tui.FocusChain
	// Viewが表示状態になったときに呼び出される。
	// 定期更新のためのworkerを起動することなどを想定。
	// Viewが破棄された、もしくは非表示状態になったときは、ctxがキャンセルされる。
	// 実行中はUIのレンダリングが止まるため、可能な限り実行時間が短い処理のみにすること。
	// なお、別途Update()メソッドが呼び出されるため、Start()メソッドの中でUpdate()メソッドを呼ぶのは推薦しない。
	Start(ctx context.Context)
}

type Controller struct {
	Config *config.Config
	Api    *restapi.Client
	LogID  string
	UI     tui.UI

	view       View
	viewCancel context.CancelFunc

	ctx    context.Context
	cancel context.CancelFunc
}

func (v *Controller) Run() error {
	var err error

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
	theme.SetStyle("label."+RunningStyleName, tui.Style{
		Fg: tui.ColorYellow,
	})
	theme.SetStyle("line.stopped", tui.Style{
		Fg: tui.ColorBlue,
	})
	theme.SetStyle("line.stopped.selected", tui.Style{
		Fg:      tui.ColorBlue,
		Bold:    tui.DecorationOn,
		Reverse: tui.DecorationOn,
	})
	theme.SetStyle("line.stopped.marked", tui.Style{
		Fg:   tui.ColorWhite,
		Bg:   tui.ColorBlue,
		Bold: tui.DecorationOn,
	})
	theme.SetStyle("line.running", tui.Style{
		Fg:   tui.ColorGreen,
		Bold: tui.DecorationOn,
	})
	theme.SetStyle("line.running.selected", tui.Style{
		Fg:      tui.ColorGreen,
		Bold:    tui.DecorationOn,
		Reverse: tui.DecorationOn,
	})
	theme.SetStyle("line.running.marked", tui.Style{
		Fg:   tui.ColorWhite,
		Bg:   tui.ColorGreen,
		Bold: tui.DecorationOn,
	})
	v.UI.SetTheme(theme)

	v.ctx, v.cancel = context.WithCancel(context.Background())
	defer v.cancel()

	view := newSelectLogView(v)
	v.setView(view)

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
	if v.view != nil {
		// stop old view.
		v.viewCancel()
	}

	v.view = view
	v.UI.SetWidget(v.view)

	// rebuild key bind settings.
	v.UI.ClearKeybindings()
	v.setKeybindings()
	v.view.SetKeybindings()

	// update focus chain
	v.UI.SetFocusChain(v.view.FocusChain())

	var viewCtx context.Context
	viewCtx, v.viewCancel = context.WithCancel(v.ctx)
	v.view.Start(viewCtx)
	go v.UI.Update(v.view.Update)
}

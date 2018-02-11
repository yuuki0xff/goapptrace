package logviewer

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/tui-go"
)

// UICoordinator implements of Coordinator.
type UICoordinator struct {
	Config *config.Config
	Api    *restapi.Client
	LogID  string
	UI     tui.UI

	m sync.Mutex

	vm       ViewModel
	vmCtx    context.Context
	vmCancel context.CancelFunc

	ctx    context.Context
	cancel context.CancelFunc
}

func (v *UICoordinator) Run() error {
	var err error

	v.UI, err = tui.New(tui.NewSpacer())
	if err != nil {
		return errors.Wrap(err, "failed to initialize TUI")
	}
	v.UI.SetTheme(v.theme())

	v.ctx, v.cancel = context.WithCancel(context.Background())
	defer v.cancel()

	go v.SetState(UIState{})
	if err := v.UI.Run(); err != nil {
		return errors.Wrap(err, "failed to initialize TUI")
	}
	return nil
}
func (v *UICoordinator) Quit() {
	v.UI.Quit()
}
func (v *UICoordinator) SetState(s UIState) {
	v.m.Lock()
	defer v.m.Unlock()

	if s.LogID == "" {
		v.setVM(&LogListVM{
			Root:   v,
			Client: v.Api.WithCtx(v.vmCtx),
		})
		return
	}

	if s.RecordID != 0 {
		v.setVM(&FuncCallDetailVM{
			Root:   v,
			Client: v.Api.WithCtx(v.vmCtx),
			LogID:  s.LogID,
			Record: s.Record,
		})
		return
	}

	if s.UseGraphView {
		// TODO: set GraphVM.
		v.setVM(nil)
	} else {
		// TODO: set RecordsListVM.
		v.setVM(nil)
	}
}
func (v *UICoordinator) NotifyVMUpdated() {
	var view View

	v.m.Lock()
	if v.vm != nil {
		view = v.vm.View()
	}
	v.m.Unlock()

	v.notifyVMUpdatedNolock(view)
}
func (v *UICoordinator) notifyVMUpdatedNolock(view View) {
	v.UI.Update(func() {
		v.UI.SetWidget(view.Widget())

		// rebuild key bind settings.
		v.UI.ClearKeybindings()
		v.setKeybindings(view.Keybindings())

		// update focus chain
		v.UI.SetFocusChain(view.FocusChain())
	})
}
func (v *UICoordinator) setKeybindings(bindings map[string]func()) {
	v.UI.SetKeybinding("Q", v.Quit)
	v.UI.SetKeybinding("Esc", v.Quit)

	for key, fn := range bindings {
		v.UI.SetKeybinding(key, fn)
	}
}
func (v *UICoordinator) stopVM() {
	if v.vm != nil {
		// stop old ViewModel.
		v.vmCancel()
	}
}
func (v *UICoordinator) setVM(vm ViewModel) {
	v.stopVM()
	v.vmCtx, v.vmCancel = context.WithCancel(v.ctx)
	v.vm = vm
	v.notifyVMUpdatedNolock(v.vm.View())
	go v.vm.Update(v.vmCtx)
}

// theme returns default themes.
func (v *UICoordinator) theme() *tui.Theme {
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
	theme.SetStyle("line.gap", tui.Style{
		Fg: tui.ColorBlue,
	})
	theme.SetStyle("line.stopped", tui.Style{
		Fg: tui.ColorGreen,
	})
	theme.SetStyle("line.stopped.selected", tui.Style{
		Fg:      tui.ColorGreen,
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
	return theme
}

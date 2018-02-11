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

func (c *UICoordinator) Run() error {
	var err error

	c.UI, err = tui.New(tui.NewSpacer())
	if err != nil {
		return errors.Wrap(err, "failed to initialize TUI")
	}
	c.UI.SetTheme(c.theme())

	c.ctx, c.cancel = context.WithCancel(context.Background())
	defer c.cancel()

	go c.SetState(UIState{})
	if err := c.UI.Run(); err != nil {
		return errors.Wrap(err, "failed to initialize TUI")
	}
	return nil
}
func (c *UICoordinator) Quit() {
	c.UI.Quit()
}
func (c *UICoordinator) SetState(s UIState) {
	c.m.Lock()
	defer c.m.Unlock()

	if s.LogID == "" {
		c.setVM(&LogListVM{
			Root:   c,
			Client: c.Api.WithCtx(c.vmCtx),
		})
		return
	}

	if s.RecordID != 0 {
		c.setVM(&FuncCallDetailVM{
			Root:   c,
			Client: c.Api.WithCtx(c.vmCtx),
			LogID:  s.LogID,
			Record: s.Record,
		})
		return
	}

	if s.UseGraphView {
		c.setVM(&GraphVM{
			Root:   c,
			Client: c.Api.WithCtx(c.vmCtx),
			LogID:  s.LogID,
		})
	} else {
		c.setVM(&LogRecordVM{
			Root:   c,
			Client: c.Api.WithCtx(c.vmCtx),
			LogID:  s.LogID,
		})
	}
}
func (c *UICoordinator) NotifyVMUpdated() {
	var view View

	c.m.Lock()
	if c.vm != nil {
		view = c.vm.View()
	}
	c.m.Unlock()

	c.notifyVMUpdatedNolock(view)
}
func (c *UICoordinator) notifyVMUpdatedNolock(view View) {
	c.UI.Update(func() {
		c.UI.SetWidget(view.Widget())

		// rebuild key bind settings.
		c.UI.ClearKeybindings()
		c.setKeybindings(view.Keybindings())

		// update focus chain
		c.UI.SetFocusChain(view.FocusChain())
	})
}
func (c *UICoordinator) setKeybindings(bindings map[string]func()) {
	c.UI.SetKeybinding("Q", c.Quit)
	c.UI.SetKeybinding("Esc", c.Quit)

	for key, fn := range bindings {
		c.UI.SetKeybinding(key, fn)
	}
}
func (c *UICoordinator) stopVM() {
	if c.vm != nil {
		// stop old ViewModel.
		c.vmCancel()
	}
}
func (c *UICoordinator) setVM(vm ViewModel) {
	c.stopVM()
	c.vmCtx, c.vmCancel = context.WithCancel(c.ctx)
	c.vm = vm
	c.notifyVMUpdatedNolock(c.vm.View())
	go c.vm.Update(c.vmCtx)
}

// theme returns default themes.
func (c *UICoordinator) theme() *tui.Theme {
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

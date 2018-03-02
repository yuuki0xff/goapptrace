package logviewer

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"sync"

	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/marcusolsson/tui-go"
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

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

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
		c.setVM(func(ctx context.Context) ViewModel {
			return &LogListVM{
				Root:   c,
				Client: c.Api.WithCtx(ctx),
			}
		})
		return
	}

	if s.RecordID != 0 {
		c.setVM(func(ctx context.Context) ViewModel {
			return &FuncCallDetailVM{
				Root:   c,
				Client: c.Api.WithCtx(ctx),
				LogID:  s.LogID,
				Record: s.Record,
			}
		})
		return
	}

	if s.UseGraphView {
		c.setVM(func(ctx context.Context) ViewModel {
			return &GraphVM{
				Root:   c,
				Client: c.Api.WithCtx(ctx),
				LogID:  s.LogID,
			}
		})
		return
	} else {
		c.setVM(func(ctx context.Context) ViewModel {
			return &LogRecordVM{
				Root:   c,
				Client: c.Api.WithCtx(ctx),
				LogID:  s.LogID,
			}
		})
		return
	}
}
func (c *UICoordinator) NotifyVMUpdated() {
	var view View

	c.m.Lock()
	defer c.m.Unlock()
	if c.vm != nil {
		view = c.vm.View()
	}
	c.setView(view)
}
func (c *UICoordinator) setView(view View) {
	// setViewは、UI threadから呼び出される可能性が高い。
	// UI thread上でc.UI.Update()を呼び出すとdead lockする問題を回避するために、
	// 別のスレッドからアップデーするようにしている。
	go c.UI.Update(func() {
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
func (c *UICoordinator) setVM(fn func(ctx context.Context) ViewModel) {
	if c.vm != nil {
		// stop old ViewModel.
		c.vmCancel()
	}

	c.vmCtx, c.vmCancel = context.WithCancel(c.ctx)
	c.vm = fn(c.vmCtx)
	c.setView(c.vm.View())
	go updateWorker(c.vmCtx, c.vm)
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

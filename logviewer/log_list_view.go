package logviewer

import (
	"context"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/tui-go"
)

// LogListVM implements ViewModel.
type LogListVM struct {
	Root   Coordinator
	Client restapi.ClientWithCtx

	m     sync.Mutex
	view  *LogListView
	state LLState
	err   error
	logs  []restapi.LogStatus

	selectedLogID string
}

func (vm *LogListVM) UpdateInterval() time.Duration {
	return 0
}
func (vm *LogListVM) Update(ctx context.Context) {
	logs, err := vm.Client.Logs()

	vm.m.Lock()
	defer vm.m.Unlock()
	vm.view = nil
	vm.state = LLWait
	vm.logs, vm.err = logs, err

	if vm.err == nil {
		// Timestamp(降順),ID(昇順)に並び替える。
		sort.Slice(logs, func(i, j int) bool {
			t1 := logs[i].Metadata.Timestamp.Unix()
			t2 := logs[j].Metadata.Timestamp.Unix()
			if t1 == t2 {
				return strings.Compare(logs[i].ID, logs[j].ID) < 0
			}
			return t1 > t2
		})
	}
}
func (vm *LogListVM) View() View {
	vm.m.Lock()
	defer vm.m.Unlock()

	if vm.view == nil {
		vm.view = &LogListView{
			VM:    vm,
			State: vm.state,
			Error: vm.err,
			Logs:  vm.logs,
		}
	}
	return vm.view
}
func (vm *LogListVM) onSelectedLog(logID string) {
	vm.selectedLogID = logID
	vm.Root.SetState(UIState{
		LogID: logID,
	})
}
func (vm *LogListVM) SelectedLog() string {
	return vm.selectedLogID
}

// LogListView implements View
type LogListView struct {
	VM    *LogListVM
	State LLState
	Error error
	Logs  []restapi.LogStatus

	initOnce sync.Once
	widget   tui.Widget
	fc       tui.FocusChain

	table *headerTable
}

func (v *LogListView) init() {
	switch v.State {
	case LLLoadingState:
		space := tui.NewSpacer()
		v.widget = tui.NewVBox(
			space,
			v.newStatusBar(LoadingText),
		)
		v.fc = newFocusChain(space)
		return
	case LLWait:
		if v.Error != nil {
			errmsg := newErrorMsg(v.Error)
			v.widget = tui.NewVBox(
				errmsg,
				tui.NewSpacer(),
				v.newStatusBar(ErrorText),
			)
			v.fc = newFocusChain(errmsg)
			return
		} else {
			v.table = v.newTable(v.Logs)
			v.widget = tui.NewVBox(
				v.table,
				tui.NewSpacer(),
				v.newStatusBar(""),
			)
			v.fc = newFocusChain(v.table)
			return
		}
	case LLSelectedState:
		// do nothing.
		space := tui.NewSpacer()
		v.widget = space
		v.fc = newFocusChain(space)
		return
	default:
		log.Panic("bug")
	}
}

func (v *LogListView) Widget() tui.Widget {
	v.initOnce.Do(v.init)
	return v.widget
}

func (v *LogListView) Keybindings() map[string]func() {
	v.initOnce.Do(v.init)
	selected := func() {
		v.onSelectedLog(nil)
	}
	return map[string]func(){
		"Right": selected,
		"l":     selected,
	}
}
func (v *LogListView) FocusChain() tui.FocusChain {
	v.initOnce.Do(v.init)
	return v.fc
}

// ログを選択したときにコールバックされる関数。
func (v *LogListView) onSelectedLog(table *tui.Table) {
	if v.table.Selected() <= 0 {
		return
	}

	idx := v.table.Selected() - 1
	id := v.Logs[idx].ID
	v.VM.onSelectedLog(id)
}

func (v *LogListView) newStatusBar(text string) *tui.StatusBar {
	s := tui.NewStatusBar(LoadingText)
	s.SetPermanentText("Log List")
	s.SetText(text)
	return s
}

func (v *LogListView) newTable(logs []restapi.LogStatus) *headerTable {
	t := newHeaderTable(
		tui.NewLabel("Status"),
		tui.NewLabel("LogID"),
		tui.NewLabel("Timestamp"),
	)
	t.OnItemActivated(v.onSelectedLog)

	for _, l := range logs {
		var status *tui.Label
		if l.ReadOnly {
			status = tui.NewLabel(StatusStoppedText)
			status.SetStyleName(StoppedStyleName)
		} else {
			status = tui.NewLabel(StatusRunningText)
			status.SetStyleName(RunningStyleName)
		}

		t.AppendRow(
			status,
			tui.NewLabel(l.ID),
			tui.NewLabel(l.Metadata.Timestamp.String()),
		)
	}
	return t
}

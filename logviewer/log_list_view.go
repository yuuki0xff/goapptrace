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

type LogListState struct {
	State      LLState
	Error      error
	Logs       []restapi.LogStatus
	SelectedID string
}
type LogListStateMutable LogListState

// LogListVM implements ViewModel.
type LogListVM struct {
	Root   Coordinator
	Client restapi.ClientWithCtx

	m     sync.Mutex
	view  *LogListView
	state LogListStateMutable
}

func (vm *LogListVM) UpdateInterval() time.Duration {
	return 0
}
func (vm *LogListVM) Update(ctx context.Context) {
	logs, err := vm.Client.Logs()
	if err == nil {
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

	vm.m.Lock()
	vm.view = nil
	vm.state.State = LLWait
	vm.state.Error = err
	vm.state.Logs = logs
	vm.m.Unlock()

	vm.Root.NotifyVMUpdated()
}
func (vm *LogListVM) View() View {
	vm.m.Lock()
	defer vm.m.Unlock()

	if vm.view == nil {
		vm.view = &LogListView{
			VM:           vm,
			LogListState: LogListState(vm.state),
		}
	}
	return vm.view
}
func (vm *LogListVM) onActivatedLog(logID string) {
	vm.Root.SetState(UIState{
		LogID: logID,
	})
}
func (vm *LogListVM) onSelectionChanged(logID string) {
	vm.m.Lock()
	vm.view = nil
	vm.state.SelectedID = logID
	vm.m.Unlock()

	vm.Root.NotifyVMUpdated()
}
func (vm *LogListVM) SelectedLog() string {
	vm.m.Lock()
	defer vm.m.Unlock()
	return vm.state.SelectedID
}

// LogListView implements View
type LogListView struct {
	VM *LogListVM
	LogListState

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
			v.table = v.newLogTable(v.Logs)
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
		v.onActivatedLog(nil)
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
func (v *LogListView) onActivatedLog(table *tui.Table) {
	if v.table.Selected() <= 0 {
		return
	}

	idx := v.table.Selected() - 1
	id := v.Logs[idx].ID
	v.VM.onActivatedLog(id)
}
func (v *LogListView) onSelectionChanged(table *tui.Table) {
	var id string
	if v.table.Selected() > 0 {
		idx := v.table.Selected() - 1
		id = v.Logs[idx].ID
	}
	v.VM.onSelectionChanged(id)
}

func (v *LogListView) newStatusBar(text string) *tui.StatusBar {
	s := tui.NewStatusBar(LoadingText)
	s.SetPermanentText("Log List")
	s.SetText(text)
	return s
}

func (v *LogListView) newLogTable(logs []restapi.LogStatus) *headerTable {
	t := newHeaderTable(
		tui.NewLabel("Status"),
		tui.NewLabel("LogID"),
		tui.NewLabel("Timestamp"),
	)

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

	if v.SelectedID != "" {
		for i, logobj := range v.Logs {
			if logobj.ID == v.SelectedID {
				// tableのidxは1から始まる
				idx := i + 1
				t.Select(idx)
				break
			}
		}
	}

	t.OnItemActivated(v.onActivatedLog)
	t.OnSelectionChanged(v.onSelectionChanged)
	return t
}

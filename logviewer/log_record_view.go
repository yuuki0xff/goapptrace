package logviewer

import (
	"context"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"golang.org/x/sync/errgroup"
)

const (
	fetchRecords    = 1000
	maxTableRecords = 1000
)

type LogRecordState struct {
	State      LRState
	Error      error
	Records    []restapi.FuncCall
	SelectedID logutil.FuncLogID
	Symbols    *logutil.Symbols
}
type LogRecordStateMutable LogRecordState

type LogRecordVM struct {
	Root   Coordinator
	Client restapi.ClientWithCtx
	LogID  string

	m     sync.Mutex
	state LogRecordStateMutable
	view  *LogRecordView
}

func (vm *LogRecordVM) UpdateInterval() time.Duration {
	return 0
}
func (vm *LogRecordVM) Update(ctx context.Context) {
	records, symbols, err := vm.fetch()

	vm.m.Lock()
	vm.view = nil
	vm.state.State = LRWait
	vm.state.Error = err
	vm.state.Records = records
	vm.state.Symbols = symbols
	vm.m.Unlock()

	vm.Root.NotifyVMUpdated()
}
func (vm *LogRecordVM) View() View {
	vm.m.Lock()
	defer vm.m.Unlock()

	if vm.view == nil {
		vm.view = &LogRecordView{
			VM:             vm,
			LogRecordState: LogRecordState(vm.state),
		}
	}
	return vm.view
}
func (vm *LogRecordVM) fetch() (
	records []restapi.FuncCall,
	symbols *logutil.Symbols,
	err error,
) {
	var eg errgroup.Group

	// get records
	eg.Go(func() error {
		records = make([]restapi.FuncCall, 0, 10000)
		ch, err := vm.Client.SearchFuncCalls(vm.LogID, restapi.SearchFuncCallParams{
			Limit:     fetchRecords,
			SortKey:   restapi.SortByEndTime,
			SortOrder: restapi.DescendingSortOrder,
		})
		if err != nil {
			return err
		}
		for fc := range ch {
			records = append(records, fc)
		}

		// 開始時刻が新しい順に並び替える
		sort.Slice(records, func(i, j int) bool {
			return records[i].StartTime > records[j].StartTime
		})
		return nil
	})
	eg.Go(func() error {
		var err error
		symbols, err = vm.Client.Symbols()
		return err
	})

	err = eg.Wait()
	return
}
func (vm *LogRecordVM) onUnselectedLog() {
	// LogIDを指定しない状態に戻す。
	vm.Root.SetState(UIState{})
}
func (vm *LogRecordVM) onActivatedRecord(record restapi.FuncCall) {
	vm.Root.SetState(UIState{
		LogID:    vm.LogID,
		RecordID: record.ID,
		Record:   record,
	})
}
func (vm *LogRecordVM) onSelectionChanged(id logutil.FuncLogID) {
	vm.m.Lock()
	vm.view = nil
	vm.state.SelectedID = id
	vm.m.Unlock()

	vm.Root.NotifyVMUpdated()
}
func (vm *LogRecordVM) onUseGraph() {
	vm.Root.SetState(UIState{
		LogID:        vm.LogID,
		UseGraphView: true,
	})
}

type LogRecordView struct {
	VM *LogRecordVM
	LogRecordState

	initOnce sync.Once
	widget   tui.Widget
	fc       tui.FocusChain

	table *headerTable
}

func (v *LogRecordView) init() {
	switch v.State {
	case LRLoadingState:
		space := tui.NewSpacer()
		v.widget = tui.NewVBox(
			space,
			v.newStatusBar(LoadingText),
		)
		v.fc = newFocusChain(space)
		return
	case LRWait:
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
			v.table = v.newRecordTable()
			v.widget = tui.NewVBox(
				v.table,
				tui.NewSpacer(),
				v.newStatusBar(""),
			)
			v.fc = newFocusChain(v.table)
			return
		}
	case LRSelectedState:
		// do nothing.
		space := tui.NewSpacer()
		v.widget = space
		v.fc = newFocusChain(space)
		return
	default:
		log.Panic("bug")
	}
}
func (v *LogRecordView) Widget() tui.Widget {
	v.initOnce.Do(v.init)
	return v.widget
}
func (v *LogRecordView) Keybindings() map[string]func() {
	v.initOnce.Do(v.init)
	unselect := func() {
		v.VM.onUnselectedLog()
	}
	selectRecord := func() {
		v.onActivatedRecord(nil)
	}
	graph := func() {
		v.VM.onUseGraph()
	}

	return map[string]func(){
		"Left":  unselect,
		"h":     unselect,
		"Right": selectRecord,
		"l":     selectRecord,
		"t":     graph,
	}
}
func (v *LogRecordView) FocusChain() tui.FocusChain {
	v.initOnce.Do(v.init)
	return v.fc
}
func (v *LogRecordView) onActivatedRecord(table *tui.Table) {
	if v.table.Selected() <= 0 {
		return
	}
	rec := v.Records[v.table.Selected()-1]
	v.VM.onActivatedRecord(rec)
}
func (v *LogRecordView) onSelectionChanged(table *tui.Table) {
	if v.table.Selected() <= 0 {
		return
	}
	idx := v.table.Selected() - 1
	id := v.Records[idx].ID
	v.VM.onSelectionChanged(id)
}
func (v *LogRecordView) newStatusBar(text string) *tui.StatusBar {
	s := tui.NewStatusBar(LoadingText)
	s.SetPermanentText("Function Call Logs")
	s.SetText(text)
	return s
}
func (v *LogRecordView) newRecordTable() *headerTable {
	t := newHeaderTable(
		tui.NewLabel("StartTime"),
		tui.NewLabel("ExecTime (ns)"),
		tui.NewLabel("GID"),
		tui.NewLabel("Module.Func:Line"),
	)
	t.SetColumnStretch(0, 5)
	t.SetColumnStretch(1, 3)
	t.SetColumnStretch(2, 1)
	t.SetColumnStretch(3, 20)

	// TODO: リファクタする
	n := len(v.Records)
	if maxTableRecords < n {
		n = maxTableRecords
	}
	records := v.Records[:n]
	for _, fc := range records {
		currentFrame := fc.Frames[0]

		fs := v.FsMap[currentFrame]
		fi := v.FMap[fs.Func]
		execTime := fc.EndTime - fc.StartTime

		t.AppendRow(
			tui.NewLabel(fc.StartTime.UnixTime().Format(config.TimestampFormat)),
			tui.NewLabel(strconv.Itoa(int(execTime))),
			tui.NewLabel(strconv.Itoa(int(fc.GID))),
			tui.NewLabel(fi.Name+":"+strconv.Itoa(int(fs.Line))),
		)
	}

	for i, fc := range records {
		if fc.ID == v.SelectedID {
			idx := i + 1
			t.Select(idx)
			break
		}
	}

	t.OnItemActivated(v.onActivatedRecord)
	t.OnSelectionChanged(v.onSelectionChanged)
	return t
}

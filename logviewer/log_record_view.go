package logviewer

import (
	"context"
	"log"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/tui-go"
	"golang.org/x/sync/errgroup"
)

const (
	fetchRecords    = 1000
	maxTableRecords = 1000
)

type LogRecordVM struct {
	Root   Coordinator
	Client restapi.ClientWithCtx
	LogID  string

	m       sync.Mutex
	view    *LogRecordView
	state   LRState
	err     error
	records []restapi.FuncCall
	fsMap   map[logutil.FuncStatusID]restapi.FuncStatusInfo
	fMap    map[logutil.FuncID]restapi.FuncInfo
}

func (vm *LogRecordVM) UpdateInterval() time.Duration {
	return 0
}
func (vm *LogRecordVM) Update(ctx context.Context) {
	// TODO: fetch records
	records, fsMap, fMap, err := vm.fetch()

	vm.m.Lock()
	defer vm.m.Lock()
	vm.view = nil
	vm.records = records
	vm.fsMap = fsMap
	vm.fMap = fMap
	vm.err = err
}
func (vm *LogRecordVM) View() View {
	vm.m.Lock()
	defer vm.m.Unlock()

	if vm.view == nil {
		vm.view = &LogRecordView{
			VM:      vm,
			State:   vm.state,
			Error:   vm.err,
			LogID:   vm.LogID,
			Records: vm.records,
			FsMap:   vm.fsMap,
			FMap:    vm.fMap,
		}
	}
	return vm.view
}
func (vm *LogRecordVM) fetch() (
	records []restapi.FuncCall,
	fsMap map[logutil.FuncStatusID]restapi.FuncStatusInfo,
	fMap map[logutil.FuncID]restapi.FuncInfo,
	err error,
) {
	var ch chan restapi.FuncCall
	records = make([]restapi.FuncCall, 0, 10000)
	ch, err = vm.Client.SearchFuncCalls(vm.LogID, restapi.SearchFuncCallParams{
		Limit:     fetchRecords,
		SortKey:   restapi.SortByEndTime,
		SortOrder: restapi.DescendingSortOrder,
	})
	if err == nil {
		return
	}

	var eg errgroup.Group

	// バックグラウンドでmetadataを取得するworkerへのリクエストを入れるチャンネル
	reqCh := make(chan logutil.FuncStatusID, 10000)
	// キャッシュ用。アクセスする前に必ずlockをすること。
	fsMap = make(map[logutil.FuncStatusID]restapi.FuncStatusInfo, 10000)
	fMap = make(map[logutil.FuncID]restapi.FuncInfo, 10000)
	var lock sync.Mutex

	eg.Go(func() error {
		// FuncCalls apiのレスポンスを受け取る
		for fc := range ch {
			records = append(records, fc)

			// metadata取得要求を出す
			currentFrame := fc.Frames[0]
			reqCh <- currentFrame
		}
		close(reqCh)

		// 開始時刻が新しい順に並び替える
		sort.Slice(records, func(i, j int) bool {
			return records[i].StartTime > records[j].StartTime
		})
		return nil
	})

	// メタデータを取得するワーカを起動する
	for i := 0; i < APIConnections; i++ {
		eg.Go(func() (err error) {
			for id := range reqCh {
				// FuncStatusInfoを取得する
				lock.Lock()
				_, ok := fsMap[id]
				lock.Unlock()
				if ok {
					continue
				}
				var fs restapi.FuncStatusInfo
				fs, err = vm.Client.FuncStatus(vm.LogID, strconv.Itoa(int(id)))
				if err != nil {
					return
				}
				lock.Lock()
				fsMap[id] = fs

				// FuncInfoを取得する
				_, ok = fMap[fs.Func]
				lock.Unlock()
				if ok {
					continue
				}
				var fi restapi.FuncInfo
				fi, err = vm.Client.Func(vm.LogID, strconv.Itoa(int(fs.Func)))
				if err != nil {
					return
				}
				lock.Lock()
				fMap[fs.Func] = fi
				lock.Unlock()
			}
			return
		})
	}
	err = eg.Wait()
	return
}
func (vm *LogRecordVM) onUnselectedLog() {
	// LogIDを指定しない状態に戻す。
	vm.Root.SetState(UIState{})
}
func (vm *LogRecordVM) onSelectedRecord(recordID logutil.FuncLogID) {
	vm.Root.SetState(UIState{
		LogID:    vm.LogID,
		RecordID: recordID,
	})
}
func (vm *LogRecordVM) onUseGraph() {
	vm.Root.SetState(UIState{
		LogID:        vm.LogID,
		UseGraphView: true,
	})
}

type LogRecordView struct {
	VM      *LogRecordVM
	State   LRState
	Error   error
	LogID   string
	Records []restapi.FuncCall
	FsMap   map[logutil.FuncStatusID]restapi.FuncStatusInfo
	FMap    map[logutil.FuncID]restapi.FuncInfo

	initOnce sync.Once
	widget   tui.Widget

	// TODO: remove fields
	table   *headerTable
	status  *tui.StatusBar
	records []restapi.FuncCall
	fc      tui.FocusChain
}

func (v *LogRecordView) init() {
	// TODO: create widgets.
	switch v.State {
	case LRLoadingState:
		v.widget = tui.NewVBox(
			tui.NewSpacer(),
			v.newStatusBar(LoadingText),
		)
		return
	case LRWait:
		if v.Error != nil {
			v.widget = tui.NewVBox(
				newErrorMsg(v.Error),
				tui.NewSpacer(),
				v.newStatusBar(ErrorText),
			)
			return
		} else {
			v.table = v.newTable()
			v.widget = tui.NewVBox(
				v.table,
				tui.NewSpacer(),
				v.newStatusBar(""),
			)
			return
		}
	case LRSelectedState:
		// do nothing.
		v.widget = tui.NewSpacer()
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
		v.onSelectedFuncCallRecord(nil)
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
func (v *LogRecordView) onSelectedFuncCallRecord(table *tui.Table) {
	if v.table.Selected() <= 0 {
		return
	}
	rec := &v.records[v.table.Selected()-1]
	v.VM.onSelectedRecord(rec.ID)
}
func (v *LogRecordView) newStatusBar(text string) *tui.StatusBar {
	s := tui.NewStatusBar(LoadingText)
	s.SetPermanentText("Function Call Logs")
	s.SetText(text)
	return s
}
func (v *LogRecordView) newTable() *headerTable {
	t := newHeaderTable(
		tui.NewLabel("StartTime"),
		tui.NewLabel("ExecTime (ns)"),
		tui.NewLabel("GID"),
		tui.NewLabel("Module.Func:Line"),
	)
	t.OnItemActivated(v.onSelectedFuncCallRecord)
	t.SetColumnStretch(0, 5)
	t.SetColumnStretch(1, 3)
	t.SetColumnStretch(2, 1)
	t.SetColumnStretch(3, 20)

	// TODO: リファクタする
	n := len(v.Records)
	if maxTableRecords < n {
		n = maxTableRecords
	}
	for _, fc := range v.Records[:n] {
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
	return t
}

package logviewer

import (
	"context"
	"sort"
	"strconv"
	"sync"

	"github.com/yuuki0xff/goapptrace/config"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/tui-go"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

type LogRecordView struct {
	LogID string
	Root  *Controller

	tui.Widget
	wrap        wrapWidget
	updateGroup singleflight.Group

	running uint32

	table   *headerTable
	status  *tui.StatusBar
	records []restapi.FuncCall
	fc      tui.FocusChain
}

func newShowLogView(logID string, root *Controller) *LogRecordView {
	v := &LogRecordView{
		LogID:  logID,
		Root:   root,
		status: tui.NewStatusBar(LoadingText),
	}
	v.status.SetPermanentText("Function Call Logs")
	v.wrap.SetWidget(tui.NewSpacer())

	fc := &tui.SimpleFocusChain{}
	fc.Set(&v.wrap)
	v.fc = fc

	v.Widget = tui.NewVBox(
		&v.wrap,
		tui.NewSpacer(),
		v.status,
	)
	return v
}
func (v *LogRecordView) SetKeybindings() {
	gotoLogList := func() {
		v.Root.setView(newSelectLogView(v.Root))
	}
	gotoDetailView := func() {
		v.onSelectedFuncCallRecord(nil)
	}
	gotoGraph := func() {
		v.Root.setView(newGraphView(v.LogID, v.Root))
	}

	v.Root.UI.SetKeybinding("Left", gotoLogList)
	v.Root.UI.SetKeybinding("h", gotoLogList)
	v.Root.UI.SetKeybinding("Right", gotoDetailView)
	v.Root.UI.SetKeybinding("l", gotoDetailView)
	v.Root.UI.SetKeybinding("t", gotoGraph)
}
func (v *LogRecordView) FocusChain() tui.FocusChain {
	return v.fc
}
func (v *LogRecordView) Start(ctx context.Context) {
	startAutoUpdateWorker(&v.running, ctx, v.Update)
}
func (v *LogRecordView) Update() {
	v.status.SetText(LoadingText)

	go v.updateGroup.Do("update", func() (interface{}, error) { // nolint: errcheck
		var err error
		table := v.newTable()
		records := make([]restapi.FuncCall, 0, 10000)

		defer v.Root.UI.Update(func() {
			v.table = table
			v.records = records

			if err != nil {
				v.wrap.SetWidget(newErrorMsg(err))
				v.status.SetText(ErrorText)
			} else {
				v.wrap.SetWidget(v.table)
				v.status.SetText("")
			}
		})

		// update contents
		func() {
			// TODO: リファクタする
			fetchRecords := int64(v.Size().Y * 5)
			maxTableRecords := v.Size().Y * 5

			var ch chan restapi.FuncCall
			ch, err = v.Root.Api.SearchFuncCalls(v.LogID, restapi.SearchFuncCallParams{
				Limit:     fetchRecords,
				SortKey:   restapi.SortByEndTime,
				SortOrder: restapi.DescendingSortOrder,
			})
			if err != nil {
				return
			}

			var eg errgroup.Group

			// バックグラウンドでmetadataを取得するworkerへのリクエストを入れるチャンネル
			reqCh := make(chan logutil.FuncStatusID, 10000)
			// キャッシュ用。アクセスする前に必ずlockをすること。
			fsMap := map[logutil.FuncStatusID]restapi.FuncStatusInfo{}
			fMap := map[logutil.FuncID]restapi.FuncInfo{}
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
						fs, err = v.Root.Api.FuncStatus(v.LogID, strconv.Itoa(int(id)))
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
						fi, err = v.Root.Api.Func(v.LogID, strconv.Itoa(int(fs.Func)))
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
			if err = eg.Wait(); err != nil {
				return
			}

			// TODO: リファクタする
			n := len(records)
			if maxTableRecords < n {
				n = maxTableRecords
			}
			for _, fc := range records[:n] {
				currentFrame := fc.Frames[0]

				fs := fsMap[currentFrame]
				fi := fMap[fs.Func]
				execTime := fc.EndTime - fc.StartTime

				table.AppendRow(
					tui.NewLabel(fc.StartTime.UnixTime().Format(config.TimestampFormat)),
					tui.NewLabel(strconv.Itoa(int(execTime))),
					tui.NewLabel(strconv.Itoa(int(fc.GID))),
					tui.NewLabel(fi.Name+":"+strconv.Itoa(int(fs.Line))),
				)
			}
		}()
		return nil, nil
	})
}
func (v *LogRecordView) onSelectedFuncCallRecord(table *tui.Table) {
	if v.table.Selected() <= 0 {
		return
	}
	rec := &v.records[v.table.Selected()-1]
	v.Root.setView(newFuncCallDetailView(v.LogID, rec, v.Root))
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
	return t
}

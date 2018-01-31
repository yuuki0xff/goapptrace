package logviewer

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/tui-go"
	"golang.org/x/sync/singleflight"
)

type LogListView struct {
	Root *Controller
	logs []restapi.LogStatus

	tui.Widget
	wrap        wrapWidget
	updateGroup singleflight.Group

	running uint32

	// ログの一覧を表示するためのテーブル
	table  *headerTable
	status *tui.StatusBar
	fc     tui.FocusChain
}

func newSelectLogView(root *Controller) *LogListView {
	v := &LogListView{
		Root:   root,
		status: tui.NewStatusBar(LoadingText),
	}
	v.status.SetPermanentText("Log List")
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
func (v *LogListView) SetKeybindings() {
	gotoLogView := func() {
		v.onSelectedLog(nil)
	}

	v.Root.UI.SetKeybinding("Right", gotoLogView)
	v.Root.UI.SetKeybinding("l", gotoLogView)
}
func (v *LogListView) FocusChain() tui.FocusChain {
	return v.fc
}
func (v *LogListView) Start(ctx context.Context) {
	startAutoUpdateWorker(&v.running, ctx, v.Update)
}

// ログ一覧を最新の状態に更新する。
func (v *LogListView) Update() {
	v.status.SetText(LoadingText)

	go v.updateGroup.Do("update", func() (interface{}, error) { // nolint: errcheck
		var err error
		var logs []restapi.LogStatus
		table := v.newTable()

		defer v.Root.UI.Update(func() {
			v.logs = logs
			v.table = table

			if err != nil {
				v.wrap.SetWidget(newErrorMsg(err))
				v.status.SetText(ErrorText)
			} else {
				v.wrap.SetWidget(v.table)
				v.status.SetText("")
			}
		})

		func() {
			logs, err = v.Root.Api.Logs()
			if err != nil {
				return
			}

			// Timestamp(降順),ID(昇順)に並び替える。
			sort.Slice(logs, func(i, j int) bool {
				t1 := logs[i].Metadata.Timestamp.Unix()
				t2 := logs[j].Metadata.Timestamp.Unix()
				if t1 == t2 {
					return strings.Compare(logs[i].ID, logs[j].ID) < 0
				}
				return t1 > t2
			})

			if len(logs) == 0 {
				err = errors.New(NoLogFiles)
			} else {
				for _, l := range logs {
					var status *tui.Label
					if l.ReadOnly {
						status = tui.NewLabel(StatusStoppedText)
						status.SetStyleName(StoppedStyleName)
					} else {
						status = tui.NewLabel(StatusRunningText)
						status.SetStyleName(RunningStyleName)
					}

					table.AppendRow(
						status,
						tui.NewLabel(l.ID),
						tui.NewLabel(l.Metadata.Timestamp.String()),
					)
				}
			}
		}()
		return nil, nil
	})
}

// ログを選択したときにコールバックされる関数。
func (v *LogListView) onSelectedLog(table *tui.Table) {
	if v.table.Selected() <= 0 {
		return
	}

	v.Root.setView(newShowLogView(
		v.logs[v.table.Selected()-1].ID,
		v.Root,
	))
}

func (v *LogListView) newTable() *headerTable {
	t := newHeaderTable(
		tui.NewLabel("Status"),
		tui.NewLabel("LogID"),
		tui.NewLabel("Timestamp"),
	)
	t.OnItemActivated(v.onSelectedLog)
	return t
}

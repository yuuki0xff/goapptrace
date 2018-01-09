package logviewer

import (
	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

type selectLogView struct {
	root *Controller
	logs []restapi.LogStatus
	// ログ表示領域に表示するWidgetを切り替えるために使用する。
	logView wrapWidget

	// ログの一覧を表示するためのテーブル
	table *headerTable
	// ログが1つも存在しないときに表示される
	noContent *tui.Label
	loading   *tui.Label
}

func (v *selectLogView) Widget() tui.Widget {
	v.table = newHeaderTable(
		tui.NewLabel("LogID"),
	)
	v.table.OnItemActivated(v.onSelectedLog)
	v.noContent = tui.NewLabel("Available logs not found")
	v.loading = tui.NewLabel("Loading...")

	v.logView = wrapWidget{
		Widget: v.loading,
	}
	layout := tui.NewVBox(
		v.table,
		tui.NewSpacer(),
	)
	return layout
}
func (v *selectLogView) SetKeybindings() {
	// do nothing
}
func (v *selectLogView) Quit() {
	// do nothing
}

// ログ一覧を最新の状態に更新する。
func (v *selectLogView) Update() {
	v.logs, _ = v.root.Api.Logs()

	v.table.RemoveRows()
	if len(v.logs) == 0 {
		v.logView.SetWidget(v.noContent)
		return
	} else {
		for _, l := range v.logs {
			v.table.AppendRow(
				tui.NewLabel(l.ID),
			)
		}
		v.logView.SetWidget(v.table)
	}
	tui.NewHBox()
}

// ログを選択したときにコールバックされる関数。
func (v *selectLogView) onSelectedLog(table *tui.Table) {
	if v.table.Selected() == 0 {
		return
	}
	v.root.setView(&showLogView{
		LogID: v.logs[v.table.Selected()-1].ID,
		root:  v.root,
	})
}

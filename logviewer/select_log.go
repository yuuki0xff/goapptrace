package logviewer

import (
	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

type selectLogView struct {
	Root *Controller
	logs []restapi.LogStatus

	tui.Widget
	// ログ表示領域に表示するWidgetを切り替えるために使用する。
	logView wrapWidget

	// ログの一覧を表示するためのテーブル
	table *headerTable
	// ログが1つも存在しないときに表示される
	noContent *tui.Label
	loading   *tui.Label
}

func newSelectLogView(root *Controller) *selectLogView {
	v := &selectLogView{
		Root: root,
		table: newHeaderTable(
			tui.NewLabel("LogID"),
		),
		noContent: tui.NewLabel("Available logs not found"),
		loading:   tui.NewLabel("Loading..."),
	}
	v.table.OnItemActivated(v.onSelectedLog)

	v.logView.SetWidget(v.loading)
	v.Widget = tui.NewVBox(
		v.table,
		tui.NewSpacer(),
	)
	return v
}
func (v *selectLogView) SetKeybindings() {
	// do nothing
}
func (v *selectLogView) Quit() {
	// do nothing
}

// ログ一覧を最新の状態に更新する。
func (v *selectLogView) Update() {
	v.logs, _ = v.Root.Api.Logs()

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
}

// ログを選択したときにコールバックされる関数。
func (v *selectLogView) onSelectedLog(table *tui.Table) {
	if v.table.Selected() == 0 {
		return
	}

	v.Root.setView(newShowLogView(
		v.logs[v.table.Selected()-1].ID,
		v.Root,
	))
}

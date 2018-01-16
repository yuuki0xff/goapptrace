package logviewer

import (
	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

type selectLogView struct {
	Root *Controller
	logs []restapi.LogStatus

	tui.Widget

	wrap wrapWidget
	// ログの一覧を表示するためのテーブル
	table  *headerTable
	status *tui.StatusBar
	fc     tui.FocusChain
}

func newSelectLogView(root *Controller) *selectLogView {
	v := &selectLogView{
		Root: root,
		table: newHeaderTable(
			tui.NewLabel("LogID"),
		),
		status: tui.NewStatusBar(LoadingText),
	}
	v.status.SetPermanentText("Log List")
	v.table.OnItemActivated(v.onSelectedLog)
	v.wrap.SetWidget(v.table)

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
func (v *selectLogView) SetKeybindings() {
	gotoLogView := func() {
		v.onSelectedLog(nil)
	}

	v.Root.UI.SetKeybinding("Right", gotoLogView)
	v.Root.UI.SetKeybinding("l", gotoLogView)
}
func (v *selectLogView) FocusChain() tui.FocusChain {
	return v.fc
}
func (v *selectLogView) Quit() {
	// do nothing
}

// ログ一覧を最新の状態に更新する。
func (v *selectLogView) Update() {
	v.status.SetText(LoadingText)

	var err error
	v.logs, err = v.Root.Api.Logs()

	if err != nil {
		v.wrap.SetWidget(newErrorMsg(err))
		v.status.SetText(ErrorText)
		return
	}

	v.table.RemoveRows()
	if len(v.logs) == 0 {
		v.wrap.SetWidget(tui.NewLabel(NoLogFiles))
		v.status.SetText("")
		return
	} else {
		for _, l := range v.logs {
			v.table.AppendRow(
				tui.NewLabel(l.ID),
			)
		}
		v.wrap.SetWidget(v.table)
		v.status.SetText("")
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

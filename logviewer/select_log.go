package logviewer

import (
	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

type selectLogView struct {
	root  *LogViewer
	table *tui.Table
	logs  []restapi.LogStatus
}

func (v *selectLogView) Widget() tui.Widget {
	v.table = tui.NewTable(0, 0)
	v.Update()
	v.table.OnItemActivated(func(table *tui.Table) {
		if v.table.Selected() == 0 {
			return
		}
		v.root.setView(&showLogView{
			LogID: v.logs[v.table.Selected()-1].ID,
			root:  v.root,
		})
	})
	v.table.OnSelectionChanged(func(table *tui.Table) {
		if len(v.logs) == 0 {
			if v.table.Selected() != 0 {
				v.table.Select(0)
			}
		} else {
			if v.table.Selected() == 0 {
				v.table.Select(1)
			}
		}
	})
	v.table.Select(1)

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
func (v *selectLogView) Update() {
	v.logs, _ = v.root.Api.Logs()

	v.table.RemoveRows()
	if len(v.logs) == 0 {
		v.table.AppendRow(
			tui.NewLabel("Available logs not found"),
		)
		return
	} else {
		v.table.AppendRow(
			tui.NewLabel("LogID"),
		)
		for _, l := range v.logs {
			v.table.AppendRow(
				tui.NewLabel(l.ID),
			)
		}
	}
}

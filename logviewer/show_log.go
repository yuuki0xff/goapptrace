package logviewer

import (
	"log"
	"strconv"

	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

type showLogView struct {
	LogID string
	root  *LogViewer

	table   *tui.Table
	records []restapi.FuncCall
}

func (v *showLogView) Widget() tui.Widget {
	v.table = tui.NewTable(0, 0)
	v.Update()
	v.table.OnItemActivated(func(table *tui.Table) {
		if v.table.Selected() == 0 {
			return
		}
		// TODO: 右サイドに、詳細パネルを表示する
	})
	v.table.OnSelectionChanged(func(table *tui.Table) {
		if len(v.records) == 0 {
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
func (v *showLogView) SetKeybindings() {
	// do nothing
}
func (v *showLogView) Quit() {
	// do nothing
}
func (v *showLogView) Update() {
	ch, err := v.root.Api.SearchFuncCalls(v.LogID, restapi.SearchFuncCallParams{})
	if err != nil {
		log.Panic(err)
	}

	// remove all rows
	v.table.RemoveRows()
	v.records = v.records[:0]

	// update contents
	v.table.AppendRow(
		tui.NewLabel("StartTime"),
		tui.NewLabel("ExecTime"),
		tui.NewLabel("GID"),
		tui.NewLabel("Module.Func:Line"),
	)
	for fc := range ch {
		v.records = append(v.records, fc)

		currentFrame := fc.Frames[0]
		fs, err := v.root.Api.FuncStatus(v.LogID, strconv.Itoa(int(currentFrame)))
		if err != nil {
			log.Panic(err)
		}
		fi, err := v.root.Api.Func(v.LogID, strconv.Itoa(int(fs.Func)))
		if err != nil {
			log.Panic(err)
		}
		execTime := fc.EndTime - fc.StartTime

		v.table.AppendRow(
			tui.NewLabel(strconv.Itoa(int(fc.StartTime))),
			tui.NewLabel(strconv.Itoa(int(execTime))),
			tui.NewLabel(strconv.Itoa(int(fc.GID))),
			tui.NewLabel(fi.Name+":"+strconv.Itoa(int(fs.Line))),
		)
	}
}
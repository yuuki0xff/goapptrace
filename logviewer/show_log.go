package logviewer

import (
	"log"
	"strconv"

	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

type showLogView struct {
	LogID string
	Root  *Controller

	tui.Widget
	logList wrapWidget

	table   *headerTable
	loading *tui.Label
	records []restapi.FuncCall
}

func newShowLogView(logID string, root *Controller) *showLogView {
	v := &showLogView{
		LogID: logID,
		Root:  root,
		table: newHeaderTable(
			tui.NewLabel("StartTime"),
			tui.NewLabel("ExecTime"),
			tui.NewLabel("GID"),
			tui.NewLabel("Module.Func:Line"),
		),
		loading: tui.NewLabel("Loading..."),
	}
	v.table.OnItemActivated(v.onSelectedFuncCallRecord)

	v.logList.SetWidget(v.loading)
	v.Widget = tui.NewVBox(
		&v.logList,
		tui.NewSpacer(),
	)
	return v
}
func (v *showLogView) SetKeybindings() {
	// do nothing
}
func (v *showLogView) Quit() {
	// do nothing
}
func (v *showLogView) Update() {
	ch, err := v.Root.Api.SearchFuncCalls(v.LogID, restapi.SearchFuncCallParams{})
	if err != nil {
		log.Panic(err)
	}

	// remove all rows
	v.table.RemoveRows()
	v.records = v.records[:0]

	// update contents
	for fc := range ch {
		v.records = append(v.records, fc)

		currentFrame := fc.Frames[0]
		fs, err := v.Root.Api.FuncStatus(v.LogID, strconv.Itoa(int(currentFrame)))
		if err != nil {
			log.Panic(err)
		}
		fi, err := v.Root.Api.Func(v.LogID, strconv.Itoa(int(fs.Func)))
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
	v.logList.SetWidget(v.table)
}
func (v *showLogView) onSelectedFuncCallRecord(table *tui.Table) {
	if v.table.Selected() == 0 {
		return
	}
	// TODO: 右サイドに、詳細パネルを表示する
}

package logviewer

import (
	"strconv"

	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/tui-go"
	"golang.org/x/sync/singleflight"
)

type showLogView struct {
	LogID string
	Root  *Controller

	tui.Widget
	wrap        wrapWidget
	updateGroup singleflight.Group

	table   *headerTable
	status  *tui.StatusBar
	records []restapi.FuncCall
	fc      tui.FocusChain
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
		status: tui.NewStatusBar(LoadingText),
	}
	v.status.SetPermanentText("Function Call Logs")
	v.table.OnItemActivated(v.onSelectedFuncCallRecord)
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
func (v *showLogView) SetKeybindings() {
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
func (v *showLogView) FocusChain() tui.FocusChain {
	return v.fc
}
func (v *showLogView) Quit() {
	// do nothing
}
func (v *showLogView) Update() {
	v.status.SetText(LoadingText)

	go v.updateGroup.Do("update", func() (interface{}, error) { // nolint: errcheck
		var err error
		table := newHeaderTable(v.table.Headers...)
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

		func() {
			var ch chan restapi.FuncCall
			ch, err = v.Root.Api.SearchFuncCalls(v.LogID, restapi.SearchFuncCallParams{})
			if err != nil {
				return
			}

			// update contents
			for fc := range ch {
				records = append(records, fc)

				currentFrame := fc.Frames[0]
				var fs restapi.FuncStatusInfo
				fs, err = v.Root.Api.FuncStatus(v.LogID, strconv.Itoa(int(currentFrame)))
				if err != nil {
					return
				}
				var fi restapi.FuncInfo
				fi, err = v.Root.Api.Func(v.LogID, strconv.Itoa(int(fs.Func)))
				if err != nil {
					return
				}
				execTime := fc.EndTime - fc.StartTime

				table.AppendRow(
					tui.NewLabel(strconv.Itoa(int(fc.StartTime))),
					tui.NewLabel(strconv.Itoa(int(execTime))),
					tui.NewLabel(strconv.Itoa(int(fc.GID))),
					tui.NewLabel(fi.Name+":"+strconv.Itoa(int(fs.Line))),
				)
			}
		}()
		return nil, nil
	})
}
func (v *showLogView) onSelectedFuncCallRecord(table *tui.Table) {
	if v.table.Selected() <= 0 {
		return
	}
	rec := &v.records[v.table.Selected()]
	v.Root.setView(newFuncCallDetailView(v.LogID, rec, v.Root))
}

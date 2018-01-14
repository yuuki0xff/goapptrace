package logviewer

import (
	"log"
	"strconv"

	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
)

type FuncCallDetailView struct {
	tui.Widget
	LogID  string
	Record *restapi.FuncCall
	Root   *Controller

	funcInfoTable *headerTable
	framesTable   *headerTable
}

func newFuncCallDetailView(logID string, record *restapi.FuncCall, root *Controller) *FuncCallDetailView {
	v := &FuncCallDetailView{
		LogID:  logID,
		Record: record,
		Root:   root,

		funcInfoTable: newHeaderTable(
			tui.NewLabel("Name"),
			tui.NewLabel("Value"),
		),
		framesTable: newHeaderTable(
			tui.NewLabel("Name"),
			tui.NewLabel("Line"),
			tui.NewLabel("PC"),
		),
	}
	v.funcInfoTable.OnItemActivated(v.onSelectedFilter)
	v.framesTable.OnItemActivated(v.onSelectedFrame)

	v.Widget = tui.NewVBox(
		v.funcInfoTable,
		v.framesTable,
		tui.NewSpacer(),
	)
	return v
}

func (v *FuncCallDetailView) Update() {
	v.funcInfoTable.RemoveRows()
	v.funcInfoTable.AppendRow(
		tui.NewLabel("GID"),
		tui.NewLabel(strconv.Itoa(int(v.Record.GID))),
	)

	v.framesTable.RemoveRows()
	for _, fs := range v.Record.Frames {
		fs, err := v.Root.Api.FuncStatus(v.LogID, strconv.Itoa(int(fs)))
		if err != nil {
			log.Panic(err)
		}
		fi, err := v.Root.Api.Func(v.LogID, strconv.Itoa(int(fs.Func)))
		if err != nil {
			log.Panic(err)
		}

		v.framesTable.AppendRow(
			tui.NewLabel(fi.Name),
			tui.NewLabel(strconv.Itoa(int(fs.Line))),
			tui.NewLabel("("+strconv.Itoa(int(fs.PC))+")"),
		)
	}

}
func (v *FuncCallDetailView) SetKeybindings() {
	// do nothing
}
func (v *FuncCallDetailView) Quit() {
	// do nothing
}

func (v *FuncCallDetailView) onSelectedFilter(*tui.Table) {
	if v.funcInfoTable.Selected() == 0 {
		return
	}
	log.Panic("not implemented")
}
func (v *FuncCallDetailView) onSelectedFrame(*tui.Table) {
	if v.framesTable.Selected() == 0 {
		return
	}
	log.Panic("not implemented")
}

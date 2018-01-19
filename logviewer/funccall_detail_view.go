package logviewer

import (
	"log"
	"strconv"

	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/tui-go"
	"golang.org/x/sync/singleflight"
)

type FuncCallDetailView struct {
	tui.Widget
	LogID  string
	Record *restapi.FuncCall
	Root   *Controller

	updateGroup singleflight.Group

	// focusChainに渡す値が常に同じになるようにしないと、クラッシュしてしまう問題を回避するため。
	funcInfoWrap wrapWidget
	framesWrap   wrapWidget

	status *tui.StatusBar
	fc     *tui.SimpleFocusChain
}

func newFuncCallDetailView(logID string, record *restapi.FuncCall, root *Controller) *FuncCallDetailView {
	v := &FuncCallDetailView{
		LogID:  logID,
		Record: record,
		Root:   root,

		status: tui.NewStatusBar(LoadingText),
		fc:     &tui.SimpleFocusChain{},
	}
	v.status.SetPermanentText("Function Call Detail")
	v.swapWidget(v.newWidgets())

	v.fc.Set(&v.funcInfoWrap, &v.framesWrap)
	v.Widget = tui.NewVBox(
		&v.funcInfoWrap,
		&v.framesWrap,
		tui.NewSpacer(),
		v.status,
	)
	return v
}

func (v *FuncCallDetailView) Update() {
	v.status.SetText(LoadingText)

	go v.updateGroup.Do("update", func() (interface{}, error) { // nolint: errcheck
		var err error
		funcInfoTable, framesTable := v.newWidgets()

		defer func() {
			if err != nil {
				//v.wrap.SetWidget(newErrorMsg(err))
				v.status.SetText(ErrorText)
			} else {
				//v.wrap.SetWidget(v.table)
				v.status.SetText("")
			}
			v.swapWidget(funcInfoTable, framesTable)
			v.Root.UI.Update(func() {})
		}()

		func() {
			funcInfoTable.AppendRow(
				tui.NewLabel("GID"),
				tui.NewLabel(strconv.Itoa(int(v.Record.GID))),
			)

			for _, fsid := range v.Record.Frames {
				var fs restapi.FuncStatusInfo
				fs, err = v.Root.Api.FuncStatus(v.LogID, strconv.Itoa(int(fsid)))
				if err != nil {
					return
				}
				var fi restapi.FuncInfo
				fi, err = v.Root.Api.Func(v.LogID, strconv.Itoa(int(fs.Func)))
				if err != nil {
					return
				}

				framesTable.AppendRow(
					tui.NewLabel(fi.Name),
					tui.NewLabel(strconv.Itoa(int(fs.Line))),
					tui.NewLabel("("+strconv.Itoa(int(fs.PC))+")"),
				)
			}
		}()
		return nil, nil
	})
}
func (v *FuncCallDetailView) SetKeybindings() {
	gotoLogView := func() {
		v.Root.setView(newShowLogView(v.LogID, v.Root))
	}

	v.Root.UI.SetKeybinding("Left", gotoLogView)
	v.Root.UI.SetKeybinding("h", gotoLogView)
}
func (v *FuncCallDetailView) FocusChain() tui.FocusChain {
	return v.fc
}
func (v *FuncCallDetailView) Quit() {
	// do nothing
}

func (v *FuncCallDetailView) onSelectedFilter(funcInfoTable *tui.Table) {
	if funcInfoTable.Selected() <= 0 {
		return
	}
	log.Panic("not implemented")
}
func (v *FuncCallDetailView) onSelectedFrame(framesTable *tui.Table) {
	if framesTable.Selected() <= 0 {
		return
	}
	log.Panic("not implemented")
}

// swapWidget swaps all widgets in the current view.
func (v *FuncCallDetailView) swapWidget(funcInfoTable *headerTable, framesTable *headerTable) {
	v.funcInfoWrap.SetWidget(funcInfoTable)
	v.framesWrap.SetWidget(framesTable)
}

// newWidgets returns new widget objects
func (v *FuncCallDetailView) newWidgets() (funcInfoTable *headerTable, framesTable *headerTable) {
	funcInfoTable = newHeaderTable(
		tui.NewLabel("Name"),
		tui.NewLabel("Value"),
	)
	framesTable = newHeaderTable(
		tui.NewLabel("Name"),
		tui.NewLabel("Line"),
		tui.NewLabel("PC"),
	)

	funcInfoTable.OnItemActivated(v.onSelectedFilter)
	framesTable.OnItemActivated(v.onSelectedFrame)
	return
}

package logviewer

import (
	"log"
	"strconv"

	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"golang.org/x/sync/singleflight"
)

type FuncCallDetailView struct {
	tui.Widget
	LogID  string
	Record *restapi.FuncCall
	Root   *Controller

	// TODO: エラーメッセージを表示できるようにする
	updateGroup singleflight.Group

	// TODO: これらのテーブルをラップする。
	// focusChainに渡す値が常に同じになるようにしないと、クラッシュしてしまう問題を回避するため。
	funcInfoTable *headerTable
	framesTable   *headerTable
	status        *tui.StatusBar
	fc            *tui.SimpleFocusChain
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
		status: tui.NewStatusBar(LoadingText),
		fc:     &tui.SimpleFocusChain{},
	}
	v.status.SetPermanentText("Function Call Detail")
	v.funcInfoTable.OnItemActivated(v.onSelectedFilter)
	v.framesTable.OnItemActivated(v.onSelectedFrame)
	v.funcInfoTable.SetFocused(true)

	v.rebuildWidget()
	return v
}

func (v *FuncCallDetailView) rebuildWidget() {
	v.fc.Set(v.funcInfoTable, v.framesTable)
	v.Widget = tui.NewVBox(
		v.funcInfoTable,
		v.framesTable,
		tui.NewSpacer(),
		v.status,
	)
}

func (v *FuncCallDetailView) Update() {
	v.status.SetText(LoadingText)

	go v.updateGroup.Do("update", func() (interface{}, error) {
		var err error
		defer func() {
			if err != nil {
				//v.wrap.SetWidget(newErrorMsg(err))
				v.status.SetText(ErrorText)
			} else {
				//v.wrap.SetWidget(v.table)
				v.status.SetText("")
			}
			v.rebuildWidget()
			v.Root.UI.Update(func() {})
		}()

		func() {
			funcInfoTable := newHeaderTable(v.funcInfoTable.Headers...)
			framesTable := newHeaderTable(v.framesTable.Headers...)

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
			v.funcInfoTable = funcInfoTable
			v.framesTable = framesTable
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

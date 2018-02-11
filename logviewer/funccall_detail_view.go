package logviewer

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/tui-go"
	"golang.org/x/sync/errgroup"
)

type FuncCallDetailVM struct {
	Root   Coordinator
	Client restapi.ClientWithCtx
	LogID  string
	Record restapi.FuncCall

	m      sync.Mutex
	view   *FuncCallDetailView
	state  FCDState
	err    error
	fsList []restapi.FuncStatusInfo
	fList  []restapi.FuncInfo
}

func (vm *FuncCallDetailVM) UpdateInterval() time.Duration {
	return 0
}
func (vm *FuncCallDetailVM) Update(ctx context.Context) {
	length := len(vm.Record.Frames)
	fsList := make([]restapi.FuncStatusInfo, length)
	fList := make([]restapi.FuncInfo, length)

	var eg errgroup.Group
	for i, fsid := range vm.Record.Frames {
		eg.Go(func() error {
			var fs restapi.FuncStatusInfo
			fs, err := vm.Client.FuncStatus(vm.LogID, strconv.Itoa(int(fsid)))
			if err != nil {
				return err
			}
			var fi restapi.FuncInfo
			fi, err = vm.Client.Func(vm.LogID, strconv.Itoa(int(fs.Func)))
			if err != nil {
				return err
			}

			fsList[i] = fs
			fList[i] = fi
			return nil
		})
	}
	err := eg.Wait()

	vm.m.Lock()
	defer vm.m.Unlock()
	vm.state = FCDWait
	vm.err = err
	if vm.err != nil {
		vm.fsList = fsList
		vm.fList = fList
	} else {
		vm.fsList = nil
		vm.fList = nil
	}
}
func (vm *FuncCallDetailVM) View() View {
	vm.m.Lock()
	defer vm.m.Unlock()

	if vm.view == nil {
		vm.view = &FuncCallDetailView{
			VM:     vm,
			State:  vm.state,
			Error:  vm.err,
			Record: vm.Record,
			FsList: vm.fsList,
			FList:  vm.fList,
		}
	}
	return vm.view
}
func (vm *FuncCallDetailVM) onUnselectedRecord(logID string) {
	vm.Root.SetState(UIState{
		LogID: logID,
	})
}

type FuncCallDetailView struct {
	VM     *FuncCallDetailVM
	State  FCDState
	Error  error
	LogID  string
	Record restapi.FuncCall
	FsList []restapi.FuncStatusInfo
	FList  []restapi.FuncInfo

	initOnce sync.Once
	widget   tui.Widget
	fc       tui.FocusChain
}

func (v *FuncCallDetailView) init() {
	switch v.State {
	case FCDLoading:
		space := tui.NewSpacer()
		v.widget = tui.NewVBox(
			space,
			v.newStatusBar(LoadingText),
		)
		v.fc = newFocusChain(space)
		return
	case FCDWait:
		if v.Error != nil {
			errmsg := newErrorMsg(v.Error)
			v.widget = tui.NewVBox(
				errmsg,
				tui.NewSpacer(),
				v.newStatusBar(ErrorText),
			)
			v.fc = newFocusChain(errmsg)
			return
		} else {
			fcInfo := tui.NewVBox(
				tui.NewLabel("Func:"),
				v.newFuncInfoTable(),
			)

			framesInfo := tui.NewVBox(
				tui.NewLabel("Call Stack:"),
				v.newFramesTable(),
			)

			v.widget = tui.NewVBox(
				fcInfo,
				framesInfo,
				tui.NewSpacer(),
				v.newStatusBar(""),
			)
			v.fc = newFocusChain(fcInfo, framesInfo)
			return
		}
	default:
		log.Panic("bug")
	}
}
func (v *FuncCallDetailView) Widget() tui.Widget {
	v.initOnce.Do(v.init)
	return v.widget
}
func (v *FuncCallDetailView) Keybindings() map[string]func() {
	v.initOnce.Do(v.init)
	unselect := func() {
		v.VM.onUnselectedRecord(v.LogID)
	}
	return map[string]func(){
		"Left": unselect,
		"h":    unselect,
	}
}
func (v *FuncCallDetailView) FocusChain() tui.FocusChain {
	v.initOnce.Do(v.init)
	return v.fc
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
func (v *FuncCallDetailView) newFuncInfoTable() *headerTable {
	t := newHeaderTable(
		tui.NewLabel("Name"),
		tui.NewLabel("Value"),
	)
	t.OnItemActivated(v.onSelectedFilter)

	t.AppendRow(
		tui.NewLabel("GID"),
		tui.NewLabel(strconv.Itoa(int(v.Record.GID))),
	)
	return t
}
func (v *FuncCallDetailView) newFramesTable() *headerTable {
	t := newHeaderTable(
		tui.NewLabel("Name"),
		tui.NewLabel("Line"),
		tui.NewLabel("PC"),
	)
	t.OnItemActivated(v.onSelectedFrame)
	t.SetColumnStretch(0, 10)
	t.SetColumnStretch(1, 1)
	t.SetColumnStretch(2, 3)

	for i := range v.FsList {
		fs := v.FsList[i]
		fi := v.FList[i]
		t.AppendRow(
			tui.NewLabel(fi.Name),
			tui.NewLabel(strconv.Itoa(int(fs.Line))),
			tui.NewLabel("("+strconv.Itoa(int(fs.PC))+")"),
		)
	}
	return t
}
func (v *FuncCallDetailView) newStatusBar(text string) *tui.StatusBar {
	s := tui.NewStatusBar(LoadingText)
	s.SetPermanentText("Function Call Detail")
	s.SetText(text)
	return s
}

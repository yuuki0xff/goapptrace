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

type FuncCallDetailState struct {
	State  FCDState
	Error  error
	FSList []restapi.FuncStatusInfo
	FList  []restapi.FuncInfo
	Record restapi.FuncCall
}
type FuncCallDetailStateMutable FuncCallDetailState
type FuncCallDetailVM struct {
	Root   Coordinator
	Client restapi.ClientWithCtx
	LogID  string
	Record restapi.FuncCall

	m     sync.Mutex
	view  *FuncCallDetailView
	state FuncCallDetailStateMutable

	updateOnce sync.Once
}

func (vm *FuncCallDetailVM) UpdateInterval() time.Duration {
	return 0
}
func (vm *FuncCallDetailVM) Update(ctx context.Context) {
	vm.updateOnce.Do(func() {
		length := len(vm.Record.Frames)
		fsList := make([]restapi.FuncStatusInfo, length)
		fList := make([]restapi.FuncInfo, length)

		var eg errgroup.Group
		fetch := func(i int) {
			eg.Go(func() error {
				fsid := vm.Record.Frames[i]
				fs, err := vm.Client.FuncStatus(vm.LogID, strconv.Itoa(int(fsid)))
				if err != nil {
					return err
				}
				fi, err := vm.Client.Func(vm.LogID, strconv.Itoa(int(fs.Func)))
				if err != nil {
					return err
				}

				fsList[i] = fs
				fList[i] = fi
				return nil
			})
		}
		for i := range vm.Record.Frames {
			fetch(i)
		}
		err := eg.Wait()

		vm.m.Lock()
		vm.view = nil
		vm.state.State = FCDWait
		vm.state.Error = err
		if err == nil {
			// no error
			vm.state.FSList = fsList
			vm.state.FList = fList
		} else {
			vm.state.FSList = nil
			vm.state.FList = nil
		}
		vm.state.Record = vm.Record
		vm.m.Unlock()

		vm.Root.NotifyVMUpdated()
	})
}
func (vm *FuncCallDetailVM) View() View {
	vm.m.Lock()
	defer vm.m.Unlock()

	if vm.view == nil {
		vm.view = &FuncCallDetailView{
			VM:                  vm,
			FuncCallDetailState: FuncCallDetailState(vm.state),
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
	VM *FuncCallDetailVM
	FuncCallDetailState

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
				tui.NewLabel("Func Info:"),
				v.newFuncInfoTable(),
			)

			framesInfo := tui.NewVBox(
				tui.NewLabel("Call Stack:"),
				v.newFramesTable(),
			)

			v.widget = tui.NewVBox(
				fcInfo,
				tui.NewLabel(""),
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
		v.VM.onUnselectedRecord(v.VM.LogID)
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

	for i := range v.FSList {
		fs := v.FSList[i]
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

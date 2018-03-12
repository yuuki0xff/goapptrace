package logviewer

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"golang.org/x/sync/errgroup"
)

type FuncCallDetailState struct {
	State FCDState
	Error error

	Record restapi.FuncCall
	Mods   []logutil.GoModule
	Funcs  []logutil.GoFunc
	Lines  []logutil.GoLine
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
		mods := make([]logutil.GoModule, length)
		funcs := make([]logutil.GoFunc, length)
		lines := make([]logutil.GoLine, length)

		var eg errgroup.Group
		fetch := func(i int) {
			pc := vm.Record.Frames[i]
			eg.Go(func() (err error) {
				mods[i], err = vm.Client.GoModule(vm.LogID, pc)
				return
			})
			eg.Go(func() (err error) {
				funcs[i], err = vm.Client.GoFunc(vm.LogID, pc)
				return
			})
			eg.Go(func() (err error) {
				lines[i], err = vm.Client.GoLine(vm.LogID, pc)
				return
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
		vm.state.Record = vm.Record
		if err == nil {
			// no error
			vm.state.Mods = mods
			vm.state.Funcs = funcs
			vm.state.Lines = lines
		} else {
			vm.state.Mods = nil
			vm.state.Funcs = nil
			vm.state.Lines = nil
		}
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

	for i := range v.Record.Frames {
		t.AppendRow(
			tui.NewLabel(v.Funcs[i].Name),
			tui.NewLabel(strconv.FormatUint(uint64(v.Lines[i].Line), 10)),
			tui.NewLabel("("+strconv.FormatUint(uint64(v.Record.Frames[i]), 10)+")"),
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

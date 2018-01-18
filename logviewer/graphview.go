package logviewer

import (
	"encoding/json"
	"image"
	"log"
	"strconv"

	"github.com/marcusolsson/tui-go"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
	"golang.org/x/sync/singleflight"
)

type GraphView struct {
	tui.Widget
	LogID string
	Root  *Controller

	updateGroup singleflight.Group

	status *tui.StatusBar
	graph  *GraphWidget
	fc     tui.FocusChain

	// 現在選択されている状態のFuncCallイベントのID
	selectedFLID logutil.FuncLogID
}

func newGraphView(logID string, root *Controller) *GraphView {
	v := &GraphView{
		LogID:  logID,
		Root:   root,
		status: tui.NewStatusBar(LoadingText),
		graph:  newGraphWidget(),
	}
	v.status.SetPermanentText("Function Call Graph")

	fc := &tui.SimpleFocusChain{}
	fc.Set(v)
	v.fc = fc
	v.Widget = tui.NewVBox(
		v.graph,
		v.status,
	)
	return v
}

func (v *GraphView) Update() {
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
			v.Root.UI.Update(func() {})
		}()

		func() {
			var ch chan restapi.FuncCall
			ch, err = v.Root.Api.SearchFuncCalls(v.LogID, restapi.SearchFuncCallParams{})
			if err != nil {
				return
			}

			var ls restapi.LogStatus
			ls, err = v.Root.Api.LogStatus(v.LogID)
			if err != nil {
				return
			}

			lines := v.buildLines(ch, v.selectedFLID, &ls.Metadata.UI)
			v.graph.SetLines(lines)
		}()
		return nil, nil
	})
}
func (v *GraphView) SetKeybindings() {
	// TODO: impl key event handlers
	up := func() {}
	right := func() {}
	down := func() {}
	left := func() {}

	v.Root.UI.SetKeybinding("k", up)
	v.Root.UI.SetKeybinding("Up", up)
	v.Root.UI.SetKeybinding("l", right)
	v.Root.UI.SetKeybinding("Right", right)
	v.Root.UI.SetKeybinding("j", down)
	v.Root.UI.SetKeybinding("Down", down)
	v.Root.UI.SetKeybinding("h", left)
	v.Root.UI.SetKeybinding("Left", left)
}
func (v *GraphView) FocusChain() tui.FocusChain {
	return v.fc
}
func (v *GraphView) Quit() {
	// do nothing
}

// buildLinesは、graphを構成する線分を構築して返す。
func (v *GraphView) buildLines(ch chan restapi.FuncCall, selectedFuncCall logutil.FuncLogID, config *storage.UIConfig) (lines []Line) {
	lines = make([]Line, 0, 1000)

	// TODO: build lines
	for fc := range ch {
		styleName := "line."
		if fc.IsEnded() {
			styleName += "stopped"
		} else {
			styleName += "running"
		}

		// TODO: fc.IDが設定されてない！？
		if fc.ID == selectedFuncCall {
			// fc is selected.
			styleName += ".selected"
		} else {
			var pinned bool
			var masked bool

			funcs := v.frames2funcs(fc.Frames)
			for _, fid := range funcs {
				if f, ok := config.Funcs[fid]; ok {
					pinned = pinned || f.Pinned
					masked = masked || f.Masked
				}
			}
			if g, ok := config.Goroutines[fc.GID]; ok {
				pinned = pinned || g.Pinned
				masked = masked || g.Masked
			}

			if masked {
				// fc should not display.
				continue
			} else if pinned {
				// fc is marked.
				styleName += ".marked"
			}
		}

		line := Line{
			Start: image.Point{
				X: int(fc.StartTime),
				Y: int(fc.GID),
			},
			Length:    int(fc.EndTime - fc.StartTime),
			Type:      VerticalLine,
			StartDeco: LineTerminationNormal,
			EndDeco:   LineTerminationNone,
			StyleName: styleName,
		}
		lines = append(lines, line)

		// print a log
		b, _ := json.Marshal(line)
		log.Println(string(b))
	}
	return lines
}

// frames2funcs converts logutil.FuncStatusID to logutil.FuncID.
func (v *GraphView) frames2funcs(frames []logutil.FuncStatusID) (funcs []logutil.FuncID) {
	for _, id := range frames {
		fs, err := v.Root.Api.FuncStatus(v.LogID, strconv.Itoa(int(id)))
		if err != nil {
			log.Panic(err)
		}
		funcs = append(funcs, fs.Func)
	}
	return
}

package logviewer

import (
	"image"

	"github.com/marcusolsson/tui-go"
)

type GraphView struct {
	tui.Widget
	LogID string
	Root  *Controller

	wrap    *wrapWidget
	graph   *GraphWidget
	loading *tui.Label
	fc      tui.FocusChain
}

func newGraphView(logID string, root *Controller) *GraphView {
	v := &GraphView{
		LogID:   logID,
		Root:    root,
		graph:   newGraphWidget(),
		loading: tui.NewLabel("Loading..."),
	}

	fc := &tui.SimpleFocusChain{}
	fc.Set(v)
	v.fc = fc
	v.wrap = &wrapWidget{
		Widget: v.loading,
	}
	v.Widget = v.wrap

	v.graph.AddLine(Line{
		Start:     image.Point{0, 0},
		Length:    5,
		Type:      VerticalLine,
		StartDeco: LineTerminationNone,
		EndDeco:   LineTerminationNormal,
		StyleName: "line.stopped",
	})
	v.graph.AddLine(Line{
		Start:     image.Point{1, 0},
		Length:    5,
		Type:      VerticalLine,
		StartDeco: LineTerminationNone,
		EndDeco:   LineTerminationNormal,
		StyleName: "line.stopped.selected",
	})
	v.graph.AddLine(Line{
		Start:     image.Point{2, 0},
		Length:    5,
		Type:      VerticalLine,
		StartDeco: LineTerminationNone,
		EndDeco:   LineTerminationNormal,
		StyleName: "line.stopped.marked",
	})
	v.graph.AddLine(Line{
		Start:     image.Point{3, 1},
		Length:    10,
		Type:      HorizontalLine,
		StartDeco: LineTerminationNone,
		EndDeco:   LineTerminationNormal,
		StyleName: "line.running",
	})
	v.graph.AddLine(Line{
		Start:     image.Point{3, 2},
		Length:    10,
		Type:      HorizontalLine,
		StartDeco: LineTerminationHighlight,
		EndDeco:   LineTerminationNone,
		StyleName: "line.running.selected",
	})
	v.graph.AddLine(Line{
		Start:     image.Point{3, 3},
		Length:    10,
		Type:      HorizontalLine,
		StartDeco: LineTerminationHighlight,
		EndDeco:   LineTerminationHighlight,
		StyleName: "line.running.marked",
	})
	return v
}

func (v *GraphView) Update() {
	// TODO: update graph widget
	v.wrap.SetWidget(v.graph)
}
func (v *GraphView) SetKeybindings() {
	// do nothing
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

package logviewer

import (
	"image"
	"log"

	"github.com/yuuki0xff/tui-go"
)

// 表示対象のWidgetを切り替えることが可能なWidget。
//
// このwidgetは、現時点ではどこからの使用されていない。
// いくつかのlinterがwarningを出してくる問題を回避するために、nolintをつけている。
type wrapWidget struct { // nolint
	Widget tui.Widget
}

func (w *wrapWidget) SetWidget(widget tui.Widget) {
	if widget == nil {
		log.Panic("widget argument should not nil")
	}
	var focused bool
	if w.Widget != nil {
		// un-focus the old widget.
		focused = w.Widget.IsFocused()
		w.Widget.SetFocused(false)

		w.copyWidgetState(widget, w.Widget)
	}

	// focus the new widget.
	w.Widget = widget
	w.Widget.SetFocused(focused)
}

func (w *wrapWidget) copyWidgetState(dest, src tui.Widget) {
	dt, ok1 := dest.(*headerTable)
	st, ok2 := src.(*headerTable)
	if ok1 && ok2 {
		dt.Select(st.Selected())
	}
}

func (w *wrapWidget) Draw(p *tui.Painter)                          { w.Widget.Draw(p) }
func (w *wrapWidget) MinSizeHint() image.Point                     { return w.Widget.MinSizeHint() }
func (w *wrapWidget) Size() image.Point                            { return w.Widget.Size() }
func (w *wrapWidget) SizeHint() image.Point                        { return w.Widget.SizeHint() }
func (w *wrapWidget) SizePolicy() (tui.SizePolicy, tui.SizePolicy) { return w.Widget.SizePolicy() }
func (w *wrapWidget) Resize(size image.Point)                      { w.Widget.Resize(size) }
func (w *wrapWidget) OnKeyEvent(ev tui.KeyEvent)                   { w.Widget.OnKeyEvent(ev) }
func (w *wrapWidget) SetFocused(b bool)                            { w.Widget.SetFocused(b) }
func (w *wrapWidget) IsFocused() bool                              { return w.Widget.IsFocused() }

package logviewer

import (
	"image"

	"github.com/marcusolsson/tui-go"
)

type ScrollMode int

const (
	ManualScrollMode ScrollMode = iota
	AutoScrollMode
)

type ScrollableWidget interface {
	tui.Widget
	PartialDraw(p *tui.Painter, p1, p2 image.Point)
}

type ScrollWidget struct {
	tui.Widget

	p1                       image.Point
	autoscrollX, autoscrollY bool
}

func (w *ScrollWidget) Scroll(x, y int) {
	w.p1 = image.Point{x, y}
}
func (w *ScrollWidget) AutoScroll(x, y bool) {
	w.autoscrollX = x
	w.autoscrollY = y
}

func (w *ScrollWidget) Draw(p *tui.Painter) {
	size := w.Size()
	hint := w.SizeHint()

	p1 := w.p1
	p2 := hint
	if w.autoscrollX {
		p1.X = size.X - hint.X
		if p1.X > 0 {
			p1.X = 0
		}
	}
	if w.autoscrollY {
		p1.Y = size.Y - hint.Y
		if p1.Y > 0 {
			p1.Y = 0
		}
	}

	p.Translate(p1.X, p1.Y)
	defer p.Restore()
	if sw, ok := w.Widget.(ScrollableWidget); ok {
		sw.PartialDraw(p, p1, p2)
	} else {
		w.Widget.Draw(p)
	}
}

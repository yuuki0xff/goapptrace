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
	ScrollArea() image.Point
	PartialDraw(p *tui.Painter, area1, area2 image.Point)
}

type ScrollWidget struct {
	ScrollableWidget

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
	// widgetの実際のサイズ
	size := w.Size()
	// scrollできるエリアのサイズ
	area := w.ScrollableWidget.ScrollArea()

	p1 := w.p1
	p2 := area
	if w.autoscrollX {
		p1.X = size.X - area.X
		if p1.X > 0 {
			p1.X = 0
		}
	}
	if w.autoscrollY {
		p1.Y = size.Y - area.Y
		if p1.Y > 0 {
			p1.Y = 0
		}
	}

	p.Translate(p1.X, p1.Y)
	defer p.Restore()
	w.ScrollableWidget.PartialDraw(p, p1, p2)
}

package logviewer

import (
	"image"
	"log"

	"github.com/yuuki0xff/tui-go"
)

type LineType int

func (l LineType) Rune() rune {
	switch l {
	case HorizontalLine:
		return '─'
	case VerticalLine:
		return '│'
	default:
		log.Panic("bug")
		panic("bug")
	}
}

type LineTermination int

func (p LineTermination) Rune(defaultRune rune) rune {
	switch p {
	case LineTerminationNone:
		return defaultRune
	case LineTerminationNormal:
		return '●'
	case LineTerminationHighlight:
		return '◎'
	default:
		log.Panic("bug")
		panic("bug")
	}
}

type Origin int

const (
	HorizontalLine LineType = iota
	VerticalLine
)
const (
	LineTerminationNormal LineTermination = iota
	LineTerminationHighlight
	LineTerminationNone
)
const (
	OriginTopLeft Origin = iota
	OriginTopRight
	OriginBottomLeft
	OriginBottomRight
)

type Line struct {
	Start     image.Point
	Length    int
	Type      LineType
	StartDeco LineTermination
	EndDeco   LineTermination
	StyleName string
}

// GraphWidget helps to painting the DAG (Directed Acyclic Graph).
type GraphWidget struct {
	tui.WidgetBase

	lines  []Line
	offset image.Point
	origin Origin
}

func newGraphWidget() *GraphWidget {
	v := &GraphWidget{}
	return v
}

func (v *GraphWidget) RemoveLines() {
	v.lines = nil
}
func (v *GraphWidget) SetLines(lines []Line) {
	v.lines = lines
}
func (v *GraphWidget) SetOffset(offset image.Point) {
	v.offset = offset
}
func (v *GraphWidget) SetOrigin(origin Origin) {
	v.origin = origin
}
func (v *GraphWidget) AddLine(line Line) {
	if line.Length <= 0 {
		log.Panic("invalid line. line length should larger than 0")
	}
	v.lines = append(v.lines, line)
}

func (v *GraphWidget) Draw(p *tui.Painter) {
	// 原点の座標を設定
	size := v.Size()
	switch v.origin {
	case OriginTopLeft:
		// do nothing
	case OriginBottomLeft:
		p.Translate(0, size.Y)
		defer p.Restore()
	case OriginTopRight:
		p.Translate(size.X, 0)
		defer p.Restore()
	case OriginBottomRight:
		p.Translate(size.X, size.Y)
		defer p.Restore()
	}

	// offsetを調整
	p.Translate(v.offset.X, v.offset.Y)
	defer p.Restore()

	// draw lines
	for _, line := range v.lines {
		if line.StyleName == "" {
			v.drawLine(line, p)
		} else {
			p.WithStyle(line.StyleName, func(painter *tui.Painter) {
				v.drawLine(line, painter)
			})
		}
	}
}
func (v *GraphWidget) drawLine(line Line, p *tui.Painter) {
	x, y := line.Start.X, line.Start.Y
	length := 0
	dx, dy := 0, 0

	switch line.Type {
	case HorizontalLine:
		dx++
	case VerticalLine:
		dy++
	default:
		log.Panic("bug")
	}

	startRune := line.StartDeco.Rune(line.Type.Rune())
	endRune := line.EndDeco.Rune(line.Type.Rune())
	p.DrawRune(x, y, startRune)
	for {
		length++
		if length >= line.Length {
			break
		}
		x += dx
		y += dy

		p.DrawRune(x, y, line.Type.Rune())
	}
	p.DrawRune(x, y, endRune)
}

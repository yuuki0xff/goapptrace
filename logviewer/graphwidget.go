package logviewer

import (
	"image"
	"log"

	"github.com/yuuki0xff/tui-go"
)

// LineTypeは線を伸ばす方向(縦 or 横)を示す型である。
// デフォルトは HorizontalLine である。
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

// LineTerminationは、線の終端の描画方法を指定する。
// デフォルトは LineTerminationNormal である。
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

// 原点座標を tui.Surface 上のどこに配置するか指定する。
// デフォルトは OriginTopLeft である。
type Origin int

const (
	// 横線であることを示す。
	// 基準点からX軸方向に線を伸ばす。
	HorizontalLine LineType = iota
	// 縦線であることを示す。
	// 基準点からY軸方向に線を伸ばす。
	VerticalLine
)
const (
	// 線の終端に"●"を描画する。
	// 線の長さが1なら、線の代わりに点だけが描画される。
	LineTerminationNormal LineTermination = iota
	// 線の終端に"◎"を描画する。
	// 線の長さが1なら、線の代わりに点だけが描画される。
	LineTerminationHighlight
	// 終端まで通常の縦棒 or 横棒が描画される。
	// 終端に点を描画しない。
	LineTerminationNone
)
const (
	// 原点は左上に配置する。
	//   0────→ X
	//   │....
	//   │....
	//   ↓
	//   Y
	OriginTopLeft Origin = iota
	// 原点を右上に配置する。
	// X軸の方向が反対にになることに注意。
	//   X ←────0
	//      ....│
	//      ....│
	//          ↓
	//          Y
	OriginTopRight
	// 原点を左下に配置する。通常の数学のグラフと同じ配置である。
	//   Y
	//   ⇡
	//   │....
	//   │....
	//   0────→ X
	OriginBottomLeft
	// 原点を右下に配置する。
	//          Y
	//          ⇡
	//      ....│
	//      ....│
	//   X ←────0
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
	size := v.Size()
	drawRune := func(x, y int, r rune) {
		// originの設定に従って座標を変換する。
		switch v.origin {
		case OriginTopLeft:
			// do nothing
		case OriginTopRight:
			x = size.X - 1 - x
		case OriginBottomLeft:
			y = size.Y - 1 - y
		case OriginBottomRight:
			x = size.X - 1 - x
			y = size.Y - 1 - y
		default:
			log.Panicf("bug: v.origin=%+v", v.origin)
		}
		p.DrawRune(x, y, r)
	}

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
	drawRune(x, y, startRune)
	for {
		length++
		if length >= line.Length {
			break
		}
		x += dx
		y += dy

		drawRune(x, y, line.Type.Rune())
	}
	drawRune(x, y, endRune)
}

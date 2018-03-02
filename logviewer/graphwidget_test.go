package logviewer

import (
	"image"
	"testing"

	"github.com/marcusolsson/tui-go"
	"github.com/stretchr/testify/assert"
)

type GraphWidgetTestCase struct {
	Name     string
	Expected string
	Size     image.Point

	Lines  []Line
	Offset image.Point
	Origin Origin
}

func TestGraphWidget_Draw(t *testing.T) {
	helper := func(tc GraphWidgetTestCase) {
		t.Run(tc.Name, func(t *testing.T) {
			a := assert.New(t)
			s := tui.NewTestSurface(tc.Size.X, tc.Size.Y)

			g := newGraphWidget()
			g.SetLines(tc.Lines)
			g.SetOffset(tc.Offset)
			g.SetOrigin(tc.Origin)
			g.Resize(tc.Size)
			g.Draw(tui.NewPainter(s, tui.NewTheme()))

			actual := s.String()
			a.Equal(tc.Expected, actual)
		})
	}

	helper(GraphWidgetTestCase{
		Name: "a-horizontal-line",
		Expected: `
..........
....●─────
..........
..........
..........
`,
		Size: image.Point{10, 5},
		Lines: []Line{
			{
				Start:  image.Point{4, 1},
				Length: 10,
			},
		},
	})
	helper(GraphWidgetTestCase{
		Name: "a-vertical-line",
		Expected: `
..........
...●......
...│......
...●......
..........
`,
		Size: image.Point{10, 5},
		Lines: []Line{
			{
				Start:  image.Point{3, 1},
				Length: 3,
				Type:   VerticalLine,
			},
		},
	})
	helper(GraphWidgetTestCase{
		Name: "short-lines",
		Expected: `
..●●......
..........
..........
..●.......
..........
`,
		Size: image.Point{10, 5},
		Lines: []Line{
			{
				Start:  image.Point{2, 0},
				Length: 2,
			}, {
				Start:  image.Point{2, 3},
				Length: 1,
			},
		},
	})
	helper(GraphWidgetTestCase{
		Name: "start-deco-none",
		Expected: `
..........
.─────●...
..........
..........
..........
`,
		Size: image.Point{10, 5},
		Lines: []Line{
			{
				Start:     image.Point{1, 1},
				Length:    6,
				StartDeco: LineTerminationNone,
			},
		},
	})
	helper(GraphWidgetTestCase{
		Name: "offset",
		Expected: `
..........
.●────●...
..........
..........
..........
`,
		Size: image.Point{10, 5},
		Lines: []Line{
			{
				Start:  image.Point{10, 10},
				Length: 6,
			},
		},
		Offset: image.Point{-9, -9},
	})
	helper(GraphWidgetTestCase{
		Name: "origin-top-right",
		Expected: `
..........
...●───●..
..........
..........
..........
`,
		Size: image.Point{10, 5},
		Lines: []Line{
			{
				Start:  image.Point{2, 1},
				Length: 5,
			},
		},
		Origin: OriginTopRight,
	})
	helper(GraphWidgetTestCase{
		Name: "origin-bottom-right",
		Expected: `
..........
..........
..........
...●───●..
..........
`,
		Size: image.Point{10, 5},
		Lines: []Line{
			{
				Start:  image.Point{2, 1},
				Length: 5,
			},
		},
		Origin: OriginBottomRight,
	})
	helper(GraphWidgetTestCase{
		Name: "origin-bottom-left",
		Expected: `
..........
..........
..........
..●───●...
..........
`,
		Size: image.Point{10, 5},
		Lines: []Line{
			{
				Start:  image.Point{2, 1},
				Length: 5,
			},
		},
		Origin: OriginBottomLeft,
	})
}

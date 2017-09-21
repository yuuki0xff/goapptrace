package render

import (
	"fmt"
	"io"
	"sort"

	"github.com/ajstarks/svgo"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

type LayoutType int

const (
	Goroutine LayoutType = iota
	FunctionCall
)

var (
	LayoutTypeNames = map[string]LayoutType{
		"goroutine":     Goroutine,
		"function-call": FunctionCall,
	}
)

type SVGRender struct {
	Log       *logutil.RawLogLoader
	StartTime logutil.Time
	EndTime   logutil.Time

	Height int
	Layout LayoutType
	Colors Colors
}

func (r *SVGRender) Render(w io.Writer) {
	r.Colors.Log = r.Log
	length := r.EndTime - r.StartTime
	grm := r.Log.TimeRangeMap.Get(r.StartTime, r.EndTime)

	canv := svg.New(w)
	canv.Start(int(length), r.Height)
	switch r.Layout {
	case Goroutine:
		gids := gids(grm)
		maxEndTime := logutil.Time(0)
		if err := grm.Walk(func(gr *logutil.Goroutine) error {
			if maxEndTime < gr.EndTime {
				maxEndTime = gr.EndTime
			}

			if gr.EndTime != logutil.NotEnded {
				var y int
				for i, gid := range gids {
					if logutil.GID(gid) == gr.GID {
						y = i * 2
						break
					}
				}

				width := gr.EndTime - gr.StartTime
				canv.Rect(
					int(gr.StartTime), y, int(width), 1,
					fmt.Sprintf(`fill="%s"`, r.Colors.GetByGoroutine(gr)),
				)
			}
			return nil
		}); err != nil {
			panic(err)
		}

		maxEndTime++
		if err := grm.Walk(func(gr *logutil.Goroutine) error {
			if gr.EndTime == logutil.NotEnded {
				var y int
				for i, gid := range gids {
					if logutil.GID(gid) == gr.GID {
						y = i * 2
						break
					}
				}

				width := maxEndTime - gr.StartTime
				canv.Rect(
					int(gr.StartTime), y, int(width), 1,
					fmt.Sprintf(`fill="%s"`, r.Colors.GetByGoroutine(gr)),
				)
			}
			return nil
		}); err != nil {
			panic(err)
		}

	case FunctionCall:
		yMap := yMapFrom(grm)
		maxEndTime := logutil.Time(0)
		if err := grm.Walk(func(gr *logutil.Goroutine) error {
			for _, fl := range gr.Records {
				if maxEndTime < fl.EndTime {
					maxEndTime = fl.EndTime
				}
				if fl.EndTime != logutil.NotEnded {
					yoffset := fl.Parents()
					width := fl.EndTime - fl.StartTime
					canv.Rect(
						int(fl.StartTime), yMap[fl.GID]+yoffset, int(width), 1,
						fmt.Sprintf(`fill="%s"`, r.Colors.GetByGoroutine(gr)),
					)
				}
			}
			return nil
		}); err != nil {
			panic(err)
		}

		maxEndTime++
		if err := grm.Walk(func(gr *logutil.Goroutine) error {
			for _, fl := range gr.Records {
				if fl.EndTime == logutil.NotEnded {
					yoffset := fl.Parents()
					width := maxEndTime - fl.StartTime
					canv.Rect(
						int(fl.StartTime), yMap[fl.GID]+yoffset, int(width), 1,
						fmt.Sprintf(`fill="%s"`, r.Colors.GetByGoroutine(gr)),
					)
				}
			}
			return nil
		}); err != nil {
			panic(err)
		}
	}
	canv.End()
}

func gids(gmap *logutil.GoroutineMap) []int {
	// NOTE: []GIDを[]intにキャストできないから、[]intにしている
	gids := []int{}
	if err := gmap.Walk(func(gr *logutil.Goroutine) error {
		gids = append(gids, int(gr.GID))
		return nil
	}); err != nil {
		panic(err)
	}
	sort.Ints(gids)
	return gids
}

// GIDからy座標に変換するマップを返す
func yMapFrom(gmap *logutil.GoroutineMap) map[logutil.GID]int {
	// key: GID
	// value: コールスタックの深さの最大値 (>=1)
	maxDepth := map[logutil.GID]int{}

	if err := gmap.Walk(func(gr *logutil.Goroutine) error {
		if _, ok := maxDepth[gr.GID]; !ok {
			maxDepth[gr.GID] = 0
		}

		for _, fl := range gr.Records {
			depth := fl.Parents()
			if maxDepth[gr.GID] < depth {
				maxDepth[gr.GID] = depth
			}
		}
		return nil
	}); err != nil {
		panic(err)
	}

	gids := gids(gmap)

	// key: GID
	// value: y座標
	yMap := map[logutil.GID]int{}
	for i := range gids {
		gid := logutil.GID(gids[i])
		y := 0
		if i > 0 {
			prevGid := logutil.GID(gids[i-1])
			if maxDepth[prevGid] == 0 {
				y = yMap[prevGid]
			} else {
				y = yMap[prevGid] + maxDepth[prevGid] + 1
			}
		}
		yMap[gid] = y
	}
	return yMap
}

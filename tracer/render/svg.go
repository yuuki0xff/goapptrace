package render

import (
	"fmt"
	"github.com/ajstarks/svgo"
	"github.com/yuuki0xff/goapptrace/tracer/log"
	"io"
	"sort"
)

type LayoutType int

const (
	Goroutine LayoutType = iota
	FunctionCall
)

type SVGRender struct {
	Log       *log.Log
	StartTime log.Time
	EndTime   log.Time

	Height int
	Layout LayoutType
	Colors Colors
}

func (r *SVGRender) Render(w io.Writer) {
	length := r.EndTime - r.StartTime
	grm := r.Log.TimeRangeMap.Get(r.StartTime, r.EndTime)

	canv := svg.New(w)
	canv.Start(int(length), r.Height)
	switch r.Layout {
	case Goroutine:
		gids := gids(grm)
		// render to goroutines
		grm.Walk(func(gr *log.Goroutine) error {
			var y int
			for i, gid := range gids {
				if log.GID(gid) == gr.GID {
					y = i * 2
					break
				}
			}

			width := gr.EndTime - gr.StartTime
			canv.Rect(
				int(gr.StartTime), y, int(width), 1,
				fmt.Sprintf(`fill="%s"`, r.Colors.GetByGoroutine(gr)),
			)
			return nil
		})

	case FunctionCall:
		yMap := yMapFrom(grm)
		maxEndTime := log.Time(0)
		grm.Walk(func(gr *log.Goroutine) error {
			for _, fl := range gr.Records {
				if maxEndTime < fl.EndTime {
					maxEndTime = fl.EndTime
				}
				if fl.EndTime != log.NotEnded {
					yoffset := fl.Parents()
					width := fl.EndTime - fl.StartTime
					canv.Rect(
						int(fl.StartTime), yMap[fl.GID]+yoffset, int(width), 1,
						fmt.Sprintf(`fill="%s"`, r.Colors.GetByGoroutine(gr)),
					)
				}
			}
			return nil
		})

		maxEndTime++
		grm.Walk(func(gr *log.Goroutine) error {
			for _, fl := range gr.Records {
				if fl.EndTime == log.NotEnded {
					yoffset := fl.Parents()
					width := maxEndTime - fl.StartTime
					canv.Rect(
						int(fl.StartTime), yMap[fl.GID]+yoffset, int(width), 1,
						fmt.Sprintf(`fill="%s"`, r.Colors.GetByGoroutine(gr)),
					)
				}
			}
			return nil
		})
	}
	canv.End()
}

func gids(gmap *log.GoroutineMap) []int {
	// NOTE: []GIDを[]intにキャストできないから、[]intにしている
	gids := []int{}
	if err := gmap.Walk(func(gr *log.Goroutine) error {
		gids = append(gids, int(gr.GID))
		return nil
	}); err != nil {
		panic(err)
	}
	sort.Ints(gids)
	return gids
}

// GIDからy座標に変換するマップを返す
func yMapFrom(gmap *log.GoroutineMap) map[log.GID]int {
	// key: GID
	// value: コールスタックの深さの最大値 (>=1)
	maxDepth := map[log.GID]int{}

	if err := gmap.Walk(func(gr *log.Goroutine) error {
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
	yMap := map[log.GID]int{}
	for i := range gids {
		gid := log.GID(gids[i])
		y := 0
		if i > 0 {
			prevGid := log.GID(gids[i-1])
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

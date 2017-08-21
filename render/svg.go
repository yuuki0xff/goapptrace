package render

import (
	"github.com/ajstarks/svgo"
	"github.com/yuuki0xff/goapptrace/log"
	"io"
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
}

func (r *SVGRender) Render(w io.Writer) {
	length := r.EndTime - r.StartTime
	grm := r.Log.TimeRangeMap.Get(r.StartTime, r.EndTime)

	canv := svg.New(w)
	canv.Start(int(length), r.Height)
	switch r.Layout {
	case Goroutine:
		// render to goroutines
		grm.Walk(func(gr *log.Goroutine) error {
			width := gr.EndTime - gr.StartTime
			canv.Rect(int(gr.StartTime), int(gr.GID), int(width), 1)
			return nil
		})

	case FunctionCall:
		// TODO: y軸方向のオフセットを調整
		maxEndTime := log.Time(0)
		grm.Walk(func(gr *log.Goroutine) error {
			for _, fl := range gr.Records {
				if maxEndTime < fl.EndTime {
					maxEndTime = fl.EndTime
				}
				width := fl.EndTime - fl.StartTime
				if fl.EndTime != log.NotEnded {
					canv.Rect(int(fl.StartTime), int(fl.GID), int(width), 1)
				}
			}
			return nil
		})

		maxEndTime++
		grm.Walk(func(gr *log.Goroutine) error {
			for _, fl := range gr.Records {
				if fl.EndTime == log.NotEnded {
					width := maxEndTime - fl.StartTime
					canv.Rect(int(fl.StartTime), int(fl.GID), int(width), 1, `color="red"`)
				}
			}
			return nil
		})
	}
	canv.End()
}

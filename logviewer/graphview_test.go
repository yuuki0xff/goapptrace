package logviewer

import (
	"image"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

func sortLines(lines []Line) []Line {
	sort.Slice(lines, func(i, j int) bool {
		sti := lines[i].StyleName
		stj := lines[j].StyleName
		si := lines[i].Start
		sj := lines[j].Start
		li := lines[i].Length
		lj := lines[j].Length

		if strings.Compare(sti, stj) < 0 {
			return true
		}
		if strings.Compare(sti, stj) == 0 {
			if si.X < sj.X {
				return true
			}
			if si.X == sj.X {
				if si.Y < sj.Y {
					return true
				}
				if si.Y == sj.Y {
					if li < lj {
						return true
					}
				}
			}
		}
		return false
	})
	return lines
}

func TestGraphVM_buildLines(t *testing.T) {
	helper := func(name string, cache GraphCache, expected []Line) {
		t.Run(name, func(t *testing.T) {
			a := assert.New(t)
			vm := GraphVM{}
			actual := vm.buildLines(&cache)

			exp := sortLines(expected)
			act := sortLines(actual)
			for i := range exp {
				t.Logf("expected[%d] = %+v", i, exp[i])
			}
			for i := range act {
				t.Logf("actual[%d] = %+v", i, act[i])
			}
			a.Equal(exp, act)
		})
	}
	symbols := func(data types.SymbolsData) *types.Symbols {
		s := &types.Symbols{}
		s.Load(data)
		return s
	}

	helper("simple",
		GraphCache{
			Symbols: symbols(types.SymbolsData{}),
			Records: []types.FuncLog{
				{
					ID:        types.FuncLogID(1),
					StartTime: 1,
					EndTime:   2,
					ParentID:  types.NotFoundParent,
					Frames:    []uintptr{100},
					GID:       0,
				},
			},
			GMap: map[types.GID]types.Goroutine{
				0: {
					GID:       0,
					StartTime: 1,
					EndTime:   2,
				},
			},
		}, []Line{
			{
				Start: image.Point{
					X: 0,
					Y: 0,
				},
				Length:    2,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNone,
				EndDeco:   LineTerminationNone,
				StyleName: "line.gap",
			}, {
				Start: image.Point{
					X: 0,
					Y: 0,
				},
				Length:    2,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNormal,
				EndDeco:   LineTerminationNormal,
				StyleName: "line.stopped",
			},
		})
	helper("multi-goroutines",
		GraphCache{
			Symbols: symbols(types.SymbolsData{}),
			Records: []types.FuncLog{
				{
					ID:        1,
					StartTime: 1,
					EndTime:   2,
					ParentID:  types.NotFoundParent,
					Frames:    []uintptr{100},
					GID:       1,
				}, {
					ID:        2,
					StartTime: 3,
					EndTime:   4,
					ParentID:  types.NotFoundParent,
					Frames:    []uintptr{200},
					GID:       2,
				},
			},
			GMap: map[types.GID]types.Goroutine{
				1: {
					GID:       1,
					StartTime: 1,
					EndTime:   2,
				},
				2: {
					GID:       2,
					StartTime: 3,
					EndTime:   4,
				},
			},
		}, []Line{
			{
				Start: image.Point{
					X: 0,
					Y: 0,
				},
				Length:    2,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNone,
				EndDeco:   LineTerminationNone,
				StyleName: "line.gap",
			}, {
				Start: image.Point{
					X: 0,
					Y: 0,
				},
				Length:    2,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNormal,
				EndDeco:   LineTerminationNormal,
				StyleName: "line.stopped",
			}, {
				Start: image.Point{
					X: 2,
					Y: 1,
				},
				Length:    2,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNone,
				EndDeco:   LineTerminationNone,
				StyleName: "line.gap",
			}, {
				Start: image.Point{
					X: 2,
					Y: 1,
				},
				Length:    2,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNormal,
				EndDeco:   LineTerminationNormal,
				StyleName: "line.stopped",
			},
		})
	helper("multi-calls",
		GraphCache{
			Symbols: symbols(types.SymbolsData{}),
			Records: []types.FuncLog{
				{
					ID:        1,
					StartTime: 1,
					EndTime:   4,
					ParentID:  types.NotFoundParent,
					Frames:    []uintptr{100},
					GID:       1,
				}, {
					ID:        2,
					StartTime: 2,
					EndTime:   3,
					ParentID:  types.NotFoundParent,
					Frames:    []uintptr{110, 200},
					GID:       1,
				},
			},
			GMap: map[types.GID]types.Goroutine{
				1: {
					GID:       1,
					StartTime: 1,
					EndTime:   4,
				},
			},
		},
		[]Line{
			{
				Start: image.Point{
					X: 0,
					Y: 0,
				},
				Length:    4,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNone,
				EndDeco:   LineTerminationNone,
				StyleName: "line.gap",
			}, {
				Start: image.Point{
					X: 0,
					Y: 0,
				},
				Length:    4,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNormal,
				EndDeco:   LineTerminationNormal,
				StyleName: "line.stopped",
			}, {
				Start: image.Point{
					X: 1,
					Y: 0,
				},
				Length:    2,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNormal,
				EndDeco:   LineTerminationNormal,
				StyleName: "line.stopped",
			},
		})
}

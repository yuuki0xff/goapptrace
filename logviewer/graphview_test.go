package logviewer

import (
	"image"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/restapi"
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

	helper("simple",
		GraphCache{
			FcList: []funcCallWithFuncIDs{
				{
					FuncCall: restapi.FuncCall{
						ID:        logutil.FuncLogID(1),
						StartTime: 1,
						EndTime:   2,
						ParentID:  logutil.NotFoundParent,
						Frames:    []logutil.FuncStatusID{1},
						GID:       0,
					},
					funcs: []logutil.FuncID{1},
				},
			},
			FsList: []restapi.FuncStatusInfo{
				{
					ID:   1,
					Func: 1,
					Line: 100,
					PC:   1234,
				},
			},
			FList: []restapi.FuncInfo{
				{
					ID:    1,
					Name:  "main.main",
					File:  "main.go",
					Entry: 1000,
				},
			},
			LogInfo: restapi.LogStatus{},
			GMap: map[logutil.GID]restapi.Goroutine{
				0: {
					GID:       0,
					StartTime: 1,
					EndTime:   2,
				},
			},
		}, []Line{
			{
				Start: image.Point{
					X: -1,
					Y: 0,
				},
				Length:    2,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNone,
				EndDeco:   LineTerminationNone,
				StyleName: "line.gap",
			}, {
				Start: image.Point{
					X: -1,
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
			FcList: []funcCallWithFuncIDs{
				{
					FuncCall: restapi.FuncCall{
						ID:        1,
						StartTime: 1,
						EndTime:   2,
						ParentID:  logutil.NotFoundParent,
						Frames:    []logutil.FuncStatusID{1},
						GID:       1,
					},
					funcs: []logutil.FuncID{1},
				}, {
					FuncCall: restapi.FuncCall{
						ID:        2,
						StartTime: 3,
						EndTime:   4,
						ParentID:  logutil.NotFoundParent,
						Frames:    []logutil.FuncStatusID{2},
						GID:       2,
					},
					funcs: []logutil.FuncID{2},
				},
			},
			FsList: []restapi.FuncStatusInfo{
				{
					ID:   1,
					Func: 1,
					Line: 10,
					PC:   1234,
				}, {
					ID:   2,
					Func: 2,
					Line: 20,
					PC:   2345,
				},
			},
			FList: []restapi.FuncInfo{
				{
					ID:    1,
					Name:  "main",
					File:  "main.go",
					Entry: 1000,
				}, {
					ID:    2,
					Name:  "foo",
					File:  "main.go",
					Entry: 2000,
				},
			},
			GMap: map[logutil.GID]restapi.Goroutine{
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
					X: -3,
					Y: 0,
				},
				Length:    2,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNone,
				EndDeco:   LineTerminationNone,
				StyleName: "line.gap",
			}, {
				Start: image.Point{
					X: -3,
					Y: 0,
				},
				Length:    2,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNormal,
				EndDeco:   LineTerminationNormal,
				StyleName: "line.stopped",
			}, {
				Start: image.Point{
					X: -1,
					Y: 1,
				},
				Length:    2,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNone,
				EndDeco:   LineTerminationNone,
				StyleName: "line.gap",
			}, {
				Start: image.Point{
					X: -1,
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
			FcList: []funcCallWithFuncIDs{
				{
					FuncCall: restapi.FuncCall{
						ID:        1,
						StartTime: 1,
						EndTime:   4,
						ParentID:  logutil.NotFoundParent,
						Frames:    []logutil.FuncStatusID{1},
						GID:       1,
					},
					funcs: []logutil.FuncID{1},
				}, {
					FuncCall: restapi.FuncCall{
						ID:        2,
						StartTime: 2,
						EndTime:   3,
						ParentID:  logutil.NotFoundParent,
						Frames:    []logutil.FuncStatusID{2, 3},
						GID:       1,
					},
					funcs: []logutil.FuncID{1, 2},
				},
			},
			FsList: []restapi.FuncStatusInfo{
				{
					ID:   1,
					Func: 1,
					Line: 10,
					PC:   1000,
				}, {
					ID:   2,
					Func: 1,
					Line: 11,
					PC:   1100,
				}, {
					ID:   3,
					Func: 2,
					Line: 20,
					PC:   2000,
				},
			},
			FList: []restapi.FuncInfo{
				{
					ID:    1,
					Name:  "main",
					File:  "main.go",
					Entry: 1000,
				}, {
					ID:    2,
					Name:  "foo",
					File:  "main.go",
					Entry: 2000,
				},
			},
			GMap: map[logutil.GID]restapi.Goroutine{
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
					X: -3,
					Y: 0,
				},
				Length:    4,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNone,
				EndDeco:   LineTerminationNone,
				StyleName: "line.gap",
			}, {
				Start: image.Point{
					X: -3,
					Y: 0,
				},
				Length:    4,
				Type:      HorizontalLine,
				StartDeco: LineTerminationNormal,
				EndDeco:   LineTerminationNormal,
				StyleName: "line.stopped",
			}, {
				Start: image.Point{
					X: -2,
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

package log

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	BufferSize           = 1 << 16
	DefaultCallstackSize = 1024
)

func (log *Log) Load(data io.Reader) error {
	log.Records = make([]*FuncLog, 0)
	log.GoroutineMap = NewGoroutineMap()
	log.TimeRangeMap = NewTimeRangeMap()

	r := bufio.NewReaderSize(data, BufferSize)
	gmap := make(map[GID][]*FuncLog)
	lineno := 0

	for {
		raw := RawLog{}
		line, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// ignore blank lines
		if len(line) == 0 {
			continue
		}

		err = json.Unmarshal(line, &raw)
		if err != nil {
			return err
		}
		raw.Time = Time(lineno)
		lineno++

		if _, ok := gmap[raw.GID]; !ok {
			// create new goroutine
			gmap[raw.GID] = make([]*FuncLog, 0, DefaultCallstackSize)
		}

		switch raw.Tag {
		case "funcStart":
			var parent *FuncLog
			if len(gmap[raw.GID]) > 0 {
				parent = gmap[raw.GID][len(gmap[raw.GID])-1]
			}
			gmap[raw.GID] = append(gmap[raw.GID], &FuncLog{
				StartTime: raw.Time,
				EndTime:   NotEnded,
				Parent:    parent,
				Frames:    raw.Frames,
				GID:       raw.GID,
			})
		case "funcEnd":
			for i := len(gmap[raw.GID]) - 1; i >= 0; i-- {
				fl := gmap[raw.GID][i]
				if compareCallee(fl, &raw) && compareCaller(fl, &raw) {
					// detect EndTime
					fl.EndTime = raw.Time
					// add to records
					log.Records = append(log.Records, fl)
					log.GoroutineMap.Add(fl)
					log.TimeRangeMap.Add(fl)

					if i != len(gmap[raw.GID])-1 {
						fmt.Printf("WARN: missing funcEnd log: %+v\n", gmap[raw.GID][i:])
					}
					// add to goroutines
					if i == 0 {
						// remove a goroutine
						delete(gmap, raw.GID)
					} else {
						// remove older FuncLog
						gmap[raw.GID] = gmap[raw.GID][:i]
					}
					break
				}
			}
		default:
			panic(errors.New(fmt.Sprintf("Unsupported tag: %s", raw.Tag)))
		}
	}

	// end-less funcs
	for gid := range gmap {
		for _, fl := range gmap[gid] {
			log.Records = append(log.Records, fl)
			log.GoroutineMap.Add(fl)
			log.TimeRangeMap.Add(fl)
		}
	}
	return nil
}

func NewTimeRange(time Time) TimeRange {
	if time == NotEnded {
		return TimeRange{NotEnded}
	}
	return TimeRange{int(time) / TimeRangeStep}
}

func (tr TimeRange) Prev() TimeRange {
	tr.rangeID--
	return tr
}

func (tr TimeRange) Next() TimeRange {
	tr.rangeID++
	return tr
}

func NewTimeRanges(startTime Time, endTime Time) []TimeRange {
	ranges := []TimeRange{}
	sid := NewTimeRange(startTime).rangeID
	eid := NewTimeRange(endTime).rangeID

	if eid == NotEnded {
		return []TimeRange{
			{NotEnded},
		}
	}

	for id := sid; id <= eid; id++ {
		ranges = append(ranges, TimeRange{id})
	}
	return ranges
}

func NewGoroutineMap() *GoroutineMap {
	return &GoroutineMap{
		m: make(map[GID]*Goroutine),
	}
}

func (gm *GoroutineMap) Add(fl *FuncLog) {
	if gr, ok := gm.m[fl.GID]; ok {
		gr.Records = append(gr.Records, fl)
		if fl.StartTime < gr.StartTime {
			gr.StartTime = fl.StartTime
		}

		if fl.EndTime == NotEnded {
			gr.EndTime = NotEnded
		} else if gr.EndTime != NotEnded && gr.EndTime < fl.EndTime {
			gr.EndTime = fl.EndTime
		}
	} else {
		gm.m[fl.GID] = &Goroutine{
			GID:       fl.GID,
			Records:   []*FuncLog{fl},
			StartTime: fl.StartTime,
			EndTime:   fl.EndTime,
		}
	}
}

func (gm *GoroutineMap) Walk(fn func(gr *Goroutine) error) error {
	for _, gr := range gm.m {
		if err := fn(gr); err != nil {
			return err
		}
	}
	return nil
}

func NewTimeRangeMap() *TimeRangeMap {
	return &TimeRangeMap{
		m: make(map[TimeRange]*GoroutineMap),
	}
}

func (trm *TimeRangeMap) Add(fl *FuncLog) {
	if fl.EndTime == NotEnded {
		// add end-less func
		tr := NewTimeRange(NotEnded)
		if _, ok := trm.m[tr]; !ok {
			trm.m[tr] = NewGoroutineMap()
		}
		trm.m[tr].Add(fl)
	} else {
		for _, tr := range NewTimeRanges(fl.StartTime, fl.EndTime) {
			if _, ok := trm.m[tr]; !ok {
				trm.m[tr] = NewGoroutineMap()
			}
			trm.m[tr].Add(fl)
		}
	}
}

func (trm *TimeRangeMap) Walk(fn func(tr TimeRange, grm *GoroutineMap) error) error {
	for tr, grm := range trm.m {
		if err := fn(tr, grm); err != nil {
			return err
		}
	}
	return nil
}

func (trm *TimeRangeMap) Get(start Time, end Time) *GoroutineMap {
	grm := NewGoroutineMap()
	timeRanges := append(NewTimeRanges(start, end), TimeRange{NotEnded})
	for _, tr := range timeRanges {
		if _, ok := trm.m[tr]; ok {
			trm.m[tr].Walk(func(gr *Goroutine) error {
				for _, fl := range gr.Records {
					grm.Add(fl)
				}
				return nil
			})
		}
	}
	return grm
}

func (fl *FuncLog) Parents() int {
	parents := 0
	f := fl
	for f.Parent != nil {
		parents++
		f = f.Parent
	}
	return parents
}

func compareCaller(fl *FuncLog, log *RawLog) bool {
	f1 := fl.Frames[1:]
	f2 := log.Frames[1:]

	if len(f1) != len(f2) {
		return false
	}
	for i := 0; i < len(f1); i++ {
		if f1[i] != f2[i] {
			return false
		}
	}
	return true
}

func compareCallee(fl *FuncLog, log *RawLog) bool {
	return fl.Frames[0].Function == log.Frames[0].Function
}

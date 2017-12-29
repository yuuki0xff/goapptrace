package logutil

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
)

const (
	DefaultCallstackSize = 1024
)

func NewTxID() TxID {
	return TxID(rand.Int63())
}

func (s *StateSimulator) Init() {
	s.Records = make([]*FuncLog, 0)
	s.GoroutineMap = NewGoroutineMap()
	s.stacks = make(map[GID][]*FuncLog)
}

func (s *StateSimulator) Next(fl RawFuncLog) {
	if _, ok := s.stacks[fl.GID]; !ok {
		// create new goroutine
		s.stacks[fl.GID] = make([]*FuncLog, 0, DefaultCallstackSize)
	}

	switch fl.Tag {
	case FuncStart:
		var parent *FuncLog
		if len(s.stacks[fl.GID]) > 0 {
			parent = s.stacks[fl.GID][len(s.stacks[fl.GID])-1]
		}
		s.stacks[fl.GID] = append(s.stacks[fl.GID], &FuncLog{
			StartTime: fl.Time,
			EndTime:   NotEnded,
			Parent:    parent,
			Frames:    fl.Frames,
			GID:       fl.GID,
		})
	case FuncEnd:
		// 最後に呼び出した関数から順番にチェックしていく。
		// 関数の終了がログに記録できなかった場合への対策。
		for i := len(s.stacks[fl.GID]) - 1; i >= 0; i-- {
			caller := s.stacks[fl.GID][i]
			if s.compareCallee(caller, &fl) && s.compareCaller(caller, &fl) {
				// caller is the caller of fl

				// detect EndTime
				caller.EndTime = fl.Time
				// add to records
				s.Records = append(s.Records, caller)
				s.GoroutineMap.Add(caller)

				if i != len(s.stacks[fl.GID])-1 {
					log.Printf("WARN: missing funcEnd log: %+v\n", s.stacks[fl.GID][i:])
				}
				// update s.stacks
				if i == 0 {
					// remove a goroutine
					delete(s.stacks, fl.GID)
				} else {
					// remove older FuncLog
					s.stacks[fl.GID] = s.stacks[fl.GID][:i]
				}
				break
			}
		}
	default:
		panic(errors.New(fmt.Sprintf("Unsupported tag: %s", fl.Tag)))
	}

	// TODO: 関数が終了しないかどうかの判定は、別の場所で行う
	// end-less funcs
	for gid := range s.stacks {
		for _, fl := range s.stacks[gid] {
			s.Records = append(s.Records, fl)
			s.GoroutineMap.Add(fl)
		}
	}
}

// TODO: シリアライズ、デシリアライズ出来るようにする
// TODO: add GobDecode([]byte) error
// TODO: add GobEncode() ([]byte, error)
// TODO: 現在の状態を取得するメソッド
// TODO: add Goroutines() []*Goroutine
// TODO: add FunctionCalls() []*Goroutine

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

		// update StartTime
		if fl.StartTime < gr.StartTime {
			gr.StartTime = fl.StartTime
		}
		// update EndTime
		if fl.EndTime == NotEnded {
			gr.EndTime = NotEnded
		} else if gr.EndTime != NotEnded && gr.EndTime < fl.EndTime {
			gr.EndTime = fl.EndTime
		}
	} else {
		// create new goroutine
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

func (fl *FuncLog) Parents() int {
	parents := 0
	f := fl
	for f.Parent != nil {
		parents++
		f = f.Parent
	}
	return parents
}

func (s StateSimulator) compareCaller(fl *FuncLog, log *RawFuncLog) bool {
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

func (s StateSimulator) compareCallee(fl *FuncLog, log *RawFuncLog) bool {
	funcID1 := s.Symbols.FuncStatus[fl.Frames[0]].Func
	funcID2 := s.Symbols.FuncStatus[log.Frames[0]].Func
	return funcID1 == funcID2
}

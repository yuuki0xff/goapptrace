package logutil

import (
	"fmt"
	"log"
)

const (
	DefaultBufferSize = 65536
)

func (s *StateSimulator) Init() {
	s.nextID = FuncLogID(0)
	s.funcLogs = make(map[FuncLogID]*FuncLog, DefaultBufferSize)
	s.txids = make(map[TxID]FuncLogID, DefaultBufferSize)
	s.stacks = make(map[GID]FuncLogID, DefaultBufferSize)
	s.goroutines = make(map[GID]*Goroutine, DefaultBufferSize)
}

// 新しいRawFuncLogを受け取り、シミュレータの状態を更新する。
func (s *StateSimulator) Next(fl RawFuncLog) {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, isExistsGID := s.goroutines[fl.GID]

	switch fl.Tag {
	case FuncStart:
		parentID := NotFoundParent
		if isExistsGID {
			parentID = s.stacks[fl.GID]
		}

		id := s.nextID
		s.nextID++

		s.funcLogs[id] = &FuncLog{
			ID:        id,
			StartTime: fl.Timestamp,
			EndTime:   NotEnded,
			ParentID:  parentID,
			Frames:    fl.Frames,
			GID:       fl.GID,
		}
		s.txids[fl.TxID] = id
		s.stacks[fl.GID] = id

		if !isExistsGID && parentID == FuncLogID(-1) {
			// 新しいgoroutineを追加
			s.goroutines[fl.GID] = &Goroutine{
				GID:       fl.GID,
				StartTime: fl.Timestamp,
				EndTime:   NotEnded,
			}
		} else if isExistsGID && parentID == FuncLogID(-1) {
			// 終了したと思っていたgoroutineが、実はまだ動いていた。
			// 動作中に変更。
			s.goroutines[fl.GID].EndTime = NotEnded
		}
	case FuncEnd:
		if !isExistsGID {
			log.Panicf("ERROR: not found goroutine: gid=%d", fl.GID)
		}

		id, ok := s.txids[fl.TxID]
		if !ok {
			log.Panicf("ERROR: not found FuncLog: txid=%d", fl.TxID)
		}

		parentID := s.funcLogs[id].ParentID

		s.funcLogs[id].EndTime = fl.Timestamp
		delete(s.txids, fl.TxID)
		s.stacks[fl.GID] = parentID

		if parentID == FuncLogID(-1) {
			// スタックが空になったので、goroutineが終了したと見なす。
			// 終了時刻を更新。
			s.goroutines[fl.GID].EndTime = fl.Timestamp
		}
	default:
		panic(fmt.Errorf("Unsupported tag: %s", fl.Tag))
	}
}

// この期間に動作していた全ての関数についてのログを返す
// 返されるログの順序は、不定である。
func (s *StateSimulator) FuncLogs() []*FuncLog {
	s.lock.RLock()
	defer s.lock.RUnlock()
	funclogs := make([]*FuncLog, len(s.funcLogs))

	var i int
	for _, fl := range s.funcLogs {
		if fl.EndTime == NotEnded {
			// flは更新される可能性があるため、コピーをしておく
			newfl := &FuncLog{}
			*newfl = *fl
			funclogs[i] = newfl
		} else {
			// 実行が終了したので、今後更新されることはない。
			// コピーをする必要なし。
			funclogs[i] = fl
		}
		i++
	}
	return funclogs
}

// この期間に動作していた全てのgoroutineについてのログを返す
func (s *StateSimulator) Goroutines() []*Goroutine {
	s.lock.RLock()
	defer s.lock.RUnlock()
	goroutines := make([]*Goroutine, len(s.goroutines))

	var i int
	for _, g := range s.goroutines {
		// gは変更される可能性があるため、コピーを取る
		newg := &Goroutine{}
		*newg = *g
		goroutines[i] = newg
		i++
	}
	return goroutines
}

// 実行が終了した関数についてのログを削除する
func (s *StateSimulator) Clear() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for id, fl := range s.funcLogs {
		if fl.EndTime == NotEnded {
			continue
		}
		delete(s.funcLogs, id)
	}
}

// StateSimulatorへの参照を返す。
// 指定したIDに対応するStateSimulatorが存在しない場合、nilを返す。
func (s *StateSimulatorStore) Get(id LogID) *StateSimulator {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.m == nil {
		return nil
	}
	return s.m[id.Hex()]
}

// 指定したIDに対応するStateSimulatorを新規作成してから返す。
func (s *StateSimulatorStore) New(id LogID) *StateSimulator {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.m == nil {
		s.m = map[string]*StateSimulator{}
	}
	if s.m[id.Hex()] != nil {
		log.Panicf("StateSimulator(LogID=%+v) is already exist", id)
	}
	simulator := &StateSimulator{}
	simulator.Init()
	s.m[id.Hex()] = simulator
	return simulator
}

// StateSimulatorをこのストアから削除する。
func (s *StateSimulatorStore) Delete(id LogID) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.m == nil {
		return
	}
	delete(s.m, id.Hex())
}

package simulator

import (
	"fmt"
	"log"

	"github.com/yuuki0xff/goapptrace/tracer/types"
)

const (
	DefaultBufferSize = 65536
)

func (s *StateSimulator) Init() {
	s.nextID = types.FuncLogID(0)
	s.funcLogs = make(map[types.FuncLogID]*types.FuncLog, DefaultBufferSize)
	s.txids = make(map[types.TxID]types.FuncLogID, DefaultBufferSize)
	s.stacks = make(map[types.GID]types.FuncLogID, DefaultBufferSize)
	s.goroutines = make(map[types.GID]*types.Goroutine, DefaultBufferSize)
}

// 新しいRawFuncLogを受け取り、シミュレータの状態を更新する。
// fl.Frames スライスの再利用をしてはいけない。
func (s *StateSimulator) Next(fl types.RawFuncLog) {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, isExistsGID := s.goroutines[fl.GID]

	switch fl.Tag {
	case types.FuncStart:
		parentID := types.NotFoundParent
		if isExistsGID {
			parentID = s.stacks[fl.GID]
		}

		id := s.nextID
		s.nextID++

		frames := types.FramesPool.Get().([]uintptr)
		frames = frames[:len(fl.Frames)]
		copy(frames, fl.Frames)
		s.funcLogs[id] = &types.FuncLog{
			ID:        id,
			StartTime: fl.Timestamp,
			EndTime:   types.NotEnded,
			ParentID:  parentID,
			Frames:    frames,
			GID:       fl.GID,
		}
		s.txids[fl.TxID] = id
		s.stacks[fl.GID] = id

		if !isExistsGID && parentID == types.FuncLogID(-1) {
			// 新しいgoroutineを追加
			s.goroutines[fl.GID] = &types.Goroutine{
				GID:       fl.GID,
				StartTime: fl.Timestamp,
				EndTime:   types.NotEnded,
			}
		} else if isExistsGID && parentID == types.FuncLogID(-1) {
			// 終了したと思っていたgoroutineが、実はまだ動いていた。
			// 動作中に変更。
			s.goroutines[fl.GID].EndTime = types.NotEnded
		}
	case types.FuncEnd:
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

		if parentID == types.FuncLogID(-1) {
			// スタックが空になったので、goroutineが終了したと見なす。
			// 終了時刻を更新。
			s.goroutines[fl.GID].EndTime = fl.Timestamp
		}
	default:
		panic(fmt.Errorf("unsupported tag: %d", fl.Tag))
	}
}

// この期間に動作していた全ての関数についてのログを返す
// 返されるログの順序は、不定である。
// needCopy==trueのときは、返されるFuncLogオブジェクトは全てコピーされ、仕様後は FuncLogPool に戻すことが可能である。
func (s *StateSimulator) FuncLogs(needCopy bool) []*types.FuncLog {
	s.lock.RLock()
	defer s.lock.RUnlock()
	funclogs := make([]*types.FuncLog, len(s.funcLogs))

	var i int
	for _, fl := range s.funcLogs {
		if needCopy {
			// 全てのフィールドをコピーする。
			newfl := types.FuncLogPool.Get().(*types.FuncLog)
			frames := newfl.Frames
			*newfl = *fl
			newfl.Frames = frames[:len(fl.Frames)]
			copy(newfl.Frames, fl.Frames)
			funclogs[i] = newfl
		} else if fl.EndTime == types.NotEnded {
			// flは更新される可能性があるため、コピーをしておく
			// なお、Framesはコピーされない。
			newfl := types.FuncLogPool.Get().(*types.FuncLog)
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
func (s *StateSimulator) Goroutines() []*types.Goroutine {
	s.lock.RLock()
	defer s.lock.RUnlock()
	goroutines := make([]*types.Goroutine, len(s.goroutines))

	var i int
	for _, g := range s.goroutines {
		// gは変更される可能性があるため、コピーを取る
		newg := &types.Goroutine{}
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
		if fl.EndTime == types.NotEnded {
			continue
		}
		delete(s.funcLogs, id)
	}
}

// StateSimulatorへの参照を返す。
// 指定したIDに対応するStateSimulatorが存在しない場合、nilを返す。
func (s *StateSimulatorStore) Get(id types.LogID) *StateSimulator {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.m == nil {
		return nil
	}
	return s.m[id.Hex()]
}

// 指定したIDに対応するStateSimulatorを新規作成してから返す。
func (s *StateSimulatorStore) New(id types.LogID) *StateSimulator {
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
func (s *StateSimulatorStore) Delete(id types.LogID) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.m == nil {
		return
	}
	delete(s.m, id.Hex())
}

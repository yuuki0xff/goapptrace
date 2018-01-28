package logutil

import (
	"log"
	"strconv"
	"strings"
)

func (f *FuncID) UnmarshalText(text []byte) error {
	id, err := strconv.ParseUint(string(text), 10, 64)
	if err != nil {
		return err
	}
	*f = FuncID(id)
	return nil
}

func (f *FuncStatusID) UnmarshalText(text []byte) error {
	id, err := strconv.ParseUint(string(text), 10, 64)
	if err != nil {
		return err
	}
	*f = FuncStatusID(id)
	return nil
}

// 初期化する。使用前に必ず呼び出すこと。
func (s *Symbols) Init() {
	s.funcs = make([]*FuncSymbol, 0)
	s.funcStatus = make([]*FuncStatus, 0)
	if s.Writable {
		s.name2FuncID = make(map[string]FuncID)
		s.pc2FSID = make(map[uintptr]FuncStatusID)
	}
}

// 指定した状態で初期化する。
// Init()を呼び出す必要はない。
func (s *Symbols) Load(funcs []*FuncSymbol, funcStatus []*FuncStatus) {
	s.Init()
	s.funcs = funcs
	s.funcStatus = funcStatus
}

// 現在保持している全てのFuncSymbolとFuncStatusのsliceをコールバックする。
// fnの内部でファイルへの書き出しなどの処理を行うこと。
// fnに渡された引数の参照先は、fn実行終了後は非同期的に変更される可能性がある。
// fnの外部で使用する場合は、全てのオブジェクトをコピーすること。
func (s *Symbols) Save(fn func(funcs []*FuncSymbol, funcStatus []*FuncStatus) error) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return fn(s.funcs, s.funcStatus)
}

// FuncIDに対応するFuncSymbolを返す。
func (s *Symbols) Func(id FuncID) (FuncSymbol, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if FuncID(len(s.funcs)) <= id {
		return FuncSymbol{}, false
	}
	f := s.funcs[id]
	if f == nil {
		return FuncSymbol{}, false
	}
	return *f, true
}

// FuncStatusIDに対応するFuncStatusを返す。
func (s *Symbols) FuncStatus(id FuncStatusID) (FuncStatus, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if FuncStatusID(len(s.funcStatus)) <= id {
		return FuncStatus{}, false
	}
	fs := s.funcStatus[id]
	if fs == nil {
		return FuncStatus{}, false
	}
	return *fs, true
}

// 登録済みのFuncの数を返す。
func (s *Symbols) FuncsSize() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.funcs)
}

// 登録済みのFuncStatusの数を返す。
func (s *Symbols) FuncStatusSize() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.funcStatus)
}

// 登録済みの全てのFuncをコールバックする。。
// fnがエラーを返すと、中断する。
func (s *Symbols) WalkFuncs(fn func(fs FuncSymbol) error) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, fs := range s.funcs {
		if fs != nil {
			if err := fn(*fs); err != nil {
				return err
			}
		}
	}
	return nil
}

// 登録済みの全てのFuncStatusをコールバックする。
// fnがエラーを返すと、中断する。
func (s *Symbols) WalkFuncStatus(fn func(fs FuncStatus) error) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, fs := range s.funcStatus {
		if fs != nil {
			if err := fn(*fs); err != nil {
				return err
			}
		}
	}
	return nil
}

// 関数名からFuncIDを取得する.
func (s *Symbols) FuncIDFromName(name string) (id FuncID, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	id, ok = s.name2FuncID[name]
	return
}

// PC(Program Counter)の値からFuncStatusIDを取得する。
// この処理は高速で完了するので、追加済みのシンボルかどうかの判定に使用できる。
func (s *Symbols) FuncStatusIDFromPC(pc uintptr) (id FuncStatusID, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	id, ok = s.pc2FSID[pc]
	return
}

// FuncStatusIDからFuncIDを取得する。
func (s *Symbols) FuncID(id FuncStatusID) FuncID {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.funcStatus[id].Func
}

// FuncStatusIDから関数名を取得する。
func (s *Symbols) FuncName(id FuncStatusID) string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.funcs[s.funcStatus[id].Func].Name
}

// FuncStatusIDからモジュール名を返す。
func (s *Symbols) ModuleName(id FuncStatusID) string {
	funcName := s.FuncName(id)

	// strip function name from funcName
	moduleHierarchy := strings.Split(funcName, "/")
	last := len(moduleHierarchy) - 1
	moduleHierarchy[last] = strings.SplitN(moduleHierarchy[last], ".", 2)[0]
	moduleName := strings.Join(moduleHierarchy, "/")

	return moduleName
}

// diffからシンボルを一括追加する。
// 注意: KeepIDがfalseのときは、FuncIDやFuncStatusIDのIDは引き継がれない。
func (s *Symbols) AddSymbolsDiff(diff *SymbolsDiff) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, fsymbol := range diff.Funcs {
		s.addFuncNolock(fsymbol)
	}
	for _, fsatus := range diff.FuncStatus {
		s.addFuncStatusNolock(fsatus)
	}
}

// Funcを追加する。
// 同一のFuncが既に存在する場合、一致したFunc.IDとadded=falseを返す。
// IDが衝突した場合の動作は不定。
func (s *Symbols) AddFunc(symbol *FuncSymbol) (id FuncID, added bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.addFuncNolock(symbol)
}

func (s *Symbols) addFuncNolock(symbol *FuncSymbol) (id FuncID, added bool) {
	if !s.Writable {
		log.Panic("Symbols is not writable")
	}

	id, ok := s.name2FuncID[symbol.Name]
	if ok {
		// if exists, nothing to do
		return id, false
	}

	if s.KeepID {
		// symbol.IDの値が、配列の長さを超えている場合、配列の長さを伸ばす。
		for symbol.ID >= FuncID(len(s.funcs)) {
			s.funcs = append(s.funcs, nil)
		}
	} else {
		symbol.ID = FuncID(len(s.funcs))
		// increase length of the funcs array
		s.funcs = append(s.funcs, nil)
	}
	s.funcs[symbol.ID] = symbol
	s.name2FuncID[symbol.Name] = symbol.ID
	return symbol.ID, true
}

// FuncStatusを追加する。
// 同一のFuncStatusが既に存在する場合、一致したFuncStatus.IDとadded=falseを返す。
// IDが衝突した場合の動作は不定。
func (s *Symbols) AddFuncStatus(status *FuncStatus) (id FuncStatusID, added bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.addFuncStatusNolock(status)
}

func (s *Symbols) addFuncStatusNolock(status *FuncStatus) (id FuncStatusID, added bool) {
	if !s.Writable {
		log.Panic("Symbols is not writable")
	}

	id, ok := s.pc2FSID[status.PC]
	if ok {
		// if exists, nothing to do
		return id, false
	}

	if s.KeepID {
		// status.IDの値が配列の長さを超えている場合、配列の長さを伸ばす。
		for status.ID >= FuncStatusID(len(s.funcStatus)) {
			s.funcStatus = append(s.funcStatus, nil)
		}
	} else {
		status.ID = FuncStatusID(len(s.funcStatus))
		// increase length of the FuncStatus array
		s.funcStatus = append(s.funcStatus, status)
	}
	s.funcStatus[status.ID] = status
	s.pc2FSID[status.PC] = status.ID
	return status.ID, true
}

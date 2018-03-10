package logutil

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var ErrReadOnly = errors.New("read only")

func (f *FuncID) UnmarshalText(text []byte) error {
	id, err := strconv.ParseUint(string(text), 10, 64)
	if err != nil {
		return err
	}
	*f = FuncID(id)
	return nil
}

func (f *GoLineID) UnmarshalText(text []byte) error {
	id, err := strconv.ParseUint(string(text), 10, 64)
	if err != nil {
		return err
	}
	*f = GoLineID(id)
	return nil
}

// 初期化する。使用前に必ず呼び出すこと。
func (s *Symbols) Init() {
	s.funcs = make([]*GoFunc, 0)
	s.goLine = make([]*GoLine, 0)
	s.name2FuncID = make(map[string]FuncID)
	s.pc2FSID = make(map[uintptr]GoLineID)
}

// 指定した状態で初期化する。
// Init()を呼び出す必要はない。
func (s *Symbols) Load(data SymbolsData) {
	s.funcs = data.Funcs
	s.goLine = data.GoLine
	s.name2FuncID = make(map[string]FuncID)
	s.pc2FSID = make(map[uintptr]GoLineID)

	for _, f := range s.funcs {
		s.name2FuncID[f.Name] = f.ID
	}
	for _, fs := range s.goLine {
		s.pc2FSID[fs.PC] = fs.ID
	}
}

// 現在保持している全てのGoFuncとGoLineのsliceをコールバックする。
// fnの内部でファイルへの書き出しなどの処理を行うこと。
// fnに渡された引数の参照先は、fn実行終了後は非同期的に変更される可能性がある。
// fnの外部で使用する場合は、全てのオブジェクトをコピーすること。
func (s *Symbols) Save(fn SymbolsWriteFn) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return fn(SymbolsData{
		Funcs:  s.funcs,
		GoLine: s.goLine,
	})
}

// FuncIDに対応するGoFuncを返す。
func (s *Symbols) Func(id FuncID) (GoFunc, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if FuncID(len(s.funcs)) <= id {
		return GoFunc{}, false
	}
	f := s.funcs[id]
	if f == nil {
		return GoFunc{}, false
	}
	return *f, true
}

// GoLineIDに対応するGoLineを返す。
func (s *Symbols) GoLine(id GoLineID) (GoLine, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if GoLineID(len(s.goLine)) <= id {
		return GoLine{}, false
	}
	fs := s.goLine[id]
	if fs == nil {
		return GoLine{}, false
	}
	return *fs, true
}

// 登録済みのFuncの数を返す。
func (s *Symbols) FuncsSize() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.funcs)
}

// 登録済みのGoLineの数を返す。
func (s *Symbols) GoLineSize() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.goLine)
}

// 登録済みの全てのFuncをコールバックする。。
// fnがエラーを返すと、中断する。
func (s *Symbols) WalkFuncs(fn func(fs GoFunc) error) error {
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

// 登録済みの全てのGoLineをコールバックする。
// fnがエラーを返すと、中断する。
func (s *Symbols) WalkGoLine(fn func(fs GoLine) error) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	for _, fs := range s.goLine {
		if fs != nil {
			if err := fn(*fs); err != nil {
				return err
			}
		}
	}
	return nil
}

// 関数名からFuncIDを取得する.
// この処理は高速で完了するので、追加済みのシンボルかどうかの判定に使用できる。
//go:nosplit
func (s *Symbols) FuncIDFromName(name string) (id FuncID, ok bool) {
	s.lock.RLock()
	id, ok = s.name2FuncID[name]
	s.lock.RUnlock()
	return
}

// PC(Program Counter)の値からGoLineIDを取得する。
// この処理は高速で完了するので、追加済みのシンボルかどうかの判定に使用できる。
//go:nosplit
func (s *Symbols) GoLineIDFromPC(pc uintptr) (id GoLineID, ok bool) {
	s.lock.RLock()
	id, ok = s.pc2FSID[pc]
	s.lock.RUnlock()
	return
}

// GoLineIDからFuncIDを取得する。
func (s *Symbols) FuncID(id GoLineID) FuncID {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.goLine[id].Func
}

// GoLineIDから関数名を取得する。
func (s *Symbols) FuncName(id GoLineID) string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.funcs[s.goLine[id].Func].Name
}

// GoLineIDからモジュール名を返す。
func (s *Symbols) ModuleName(id GoLineID) string {
	funcName := s.FuncName(id)

	// strip function name from funcName
	moduleHierarchy := strings.Split(funcName, "/")
	last := len(moduleHierarchy) - 1
	moduleHierarchy[last] = strings.SplitN(moduleHierarchy[last], ".", 2)[0]
	moduleName := strings.Join(moduleHierarchy, "/")

	return moduleName
}

func (s *Symbols) SetSymbolsData(data *SymbolsData) {
	// TODO
}

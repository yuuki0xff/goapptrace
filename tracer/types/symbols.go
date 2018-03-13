package types

import (
	"strconv"
	"strings"
	"sync"
)

type SymbolsReadFn func() (SymbolsData, error)
type SymbolsWriteFn func(data SymbolsData) error

type Symbols struct {
	Writable bool
	// KeepIDがtrueのとき、FuncIDおよびGoLineIDは、追加時に指定されたIDを使用する。
	// KeepIDがfalseのとき、追加時に指定されたIDは無視し、新たなIDを付与する。
	KeepID bool

	lock sync.RWMutex
	data SymbolsData
}

type SymbolsData struct {
	Files []string
	Mods  []GoModule
	Funcs []GoFunc
	Lines []GoLine
}

// FileID is index of Symbols.Files array.
type FileID uint32

// File is file path to the source code.
// example: "/go/src/github.com/yuuki0xff/goapptrace/goapptrace.go"
type File string

// GoModules means a module in golang.
type GoModule struct {
	Name  string
	MinPC uintptr
	MaxPC uintptr
}

// GoFunc means a function in golang.
type GoFunc struct {
	// entry point of this function
	Entry uintptr
	// example: "github.com/yuuki0xff/goapptrace.main"
	Name string
}

// GoLine haves a correspondence to position on source code from PC (Program Counter).
type GoLine struct {
	PC uintptr
	// file location that defines this function.
	FileID FileID
	Line   uint32
}

// 初期化する。使用前に必ず呼び出すこと。
func (s *Symbols) Init() {}

// 指定した状態で初期化する。
// Init()を呼び出す必要はない。
func (s *Symbols) Load(data SymbolsData) {
	s.data = data
}

// 現在保持している全てのGoFuncとGoLineのsliceをコールバックする。
// fnの内部でファイルへの書き出しなどの処理を行うこと。
// fnに渡された引数の参照先は、fn実行終了後は非同期的に変更される可能性がある。
// fnの外部で使用する場合は、全てのオブジェクトをコピーすること。
func (s *Symbols) Save(fn SymbolsWriteFn) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return fn(s.data)
}

// pcに対応するGoModuleを返す。
func (s *Symbols) GoModule(pc uintptr) (GoModule, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, m := range s.data.Mods {
		if m.MinPC <= pc && pc <= m.MaxPC {
			// found
			return m, true
		}
	}
	// not found
	return GoModule{}, false
}

// pcに対応するGoFuncを返す。
func (s *Symbols) GoFunc(pc uintptr) (GoFunc, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// fn.Entry <= pcを満たす最後の要素を返す
	for i, fn := range s.data.Funcs {
		if fn.Entry > pc {
			if i > 0 {
				// found
				return s.data.Funcs[i-1], true
			}
			break
		}
	}
	// not found
	return GoFunc{}, false
}

// pcに対応するGoLineを返す。
func (s *Symbols) GoLine(pc uintptr) (GoLine, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// ln.PC <= pcを満たす最後の要素を返す
	for i, ln := range s.data.Lines {
		if ln.PC > pc {
			// found
			if i > 0 {
				return s.data.Lines[i-1], true
			}
			break
		}
	}
	// not found
	return GoLine{}, false
}

// 登録済みのFuncの数を返す。
func (s *Symbols) FuncsSize() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	// TODO: テストケース以外から使用されていないため、削除する
	panic("todo")
}

// 登録済みのGoLineの数を返す。
func (s *Symbols) GoLineSize() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	// TODO: テストケース以外から使用されていないため、削除する
	panic("todo")
}

// 登録済みの全てのFuncをコールバックする。。
// fnがエラーを返すと、中断する。
func (s *Symbols) WalkFuncs(fn func(fs GoFunc) error) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	// TODO: テストケース以外から使用されていない。扱いを検討する
	panic("todo")
	//for _, fs := range s.funcs {
	//	if fs != nil {
	//		if err := fn(*fs); err != nil {
	//			return err
	//		}
	//	}
	//}
	//return nil
}

// 登録済みの全てのGoLineをコールバックする。
// fnがエラーを返すと、中断する。
func (s *Symbols) WalkGoLine(fn func(fs GoLine) error) error {
	s.lock.RLock()
	defer s.lock.RUnlock()
	panic("todo")
	// TODO: テストケース以外から使用されていない。扱いを検討する
	//for _, fs := range s.goLine {
	//	if fs != nil {
	//		if err := fn(*fs); err != nil {
	//			return err
	//		}
	//	}
	//}
	//return nil
}

// 関数名からFuncIDを取得する.
// この処理は高速で完了するので、追加済みのシンボルかどうかの判定に使用できる。
//go:nosplit
func (s *Symbols) FuncIDFromName(name string) (id FuncID, ok bool) {
	s.lock.RLock()
	// TODO: logger.sendLog()の改修により不要になるため、削除する
	panic("todo")
	s.lock.RUnlock()
	return
}

// PC(Program Counter)の値からGoLineIDを取得する。
// この処理は高速で完了するので、追加済みのシンボルかどうかの判定に使用できる。
//go:nosplit
func (s *Symbols) GoLineIDFromPC(pc uintptr) (id GoLineID, ok bool) {
	s.lock.RLock()
	// TODO: logger.sendLog()の改修により不要になるため、削除する
	panic("todo")
	s.lock.RUnlock()
	return
}

// GoLineIDからFuncIDを取得する。
func (s *Symbols) FuncID(pc uintptr) FuncID {
	s.lock.RLock()
	defer s.lock.RUnlock()
	// TODO: 不要になったメソッドなので、削除する。
	panic("todo")
}

// GoLineIDから関数名を取得する。
func (s *Symbols) FuncName(pc uintptr) string {
	fn, ok := s.GoFunc(pc)
	if ok {
		return "?"
	}
	return fn.Name
}

// GoLineIDからモジュール名を返す。
func (s *Symbols) ModuleName(pc uintptr) string {
	funcName := s.FuncName(pc)

	// strip function name from funcName
	moduleHierarchy := strings.Split(funcName, "/")
	last := len(moduleHierarchy) - 1
	moduleHierarchy[last] = strings.SplitN(moduleHierarchy[last], ".", 2)[0]
	moduleName := strings.Join(moduleHierarchy, "/")

	return moduleName
}

// FileLineは、pcに対応するファイル名と行数を文字列として返す。
func (s *Symbols) FileLine(pc uintptr) string {
	filename := "?"
	linenumber := "?"
	if line, ok := s.GoLine(pc); ok {
		if int(line.FileID) <= len(s.data.Files) {
			filename = s.data.Files[line.FileID]
		}
		linenumber = strconv.FormatUint(uint64(line.Line), 10)
	}

	return filename + ":" + linenumber
}

func (s *Symbols) SetSymbolsData(data *SymbolsData) {
	s.lock.Lock()
	s.data = *data
	s.lock.Unlock()
}

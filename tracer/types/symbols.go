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
	return s.goModuleNolock(pc)
}
func (s *Symbols) goModuleNolock(pc uintptr) (GoModule, bool) {
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

	var found bool
	var idx int
	mod, ok := s.goModuleNolock(pc)
	if !ok {
		goto notFound
	}

	// fn.Entry <= pcを満たす最後の要素を返す
	for i, fn := range s.data.Funcs {
		if fn.Entry < mod.MinPC {
			continue
		}
		if fn.Entry > mod.MaxPC {
			break
		}
		if fn.Entry <= pc {
			found = true
			idx = i
		} else {
			break
		}
	}

	if found {
		return s.data.Funcs[idx], true
	}
notFound:
	return GoFunc{}, false
}

// pcに対応するGoLineを返す。
func (s *Symbols) GoLine(pc uintptr) (GoLine, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var found bool
	var idx int
	mod, ok := s.goModuleNolock(pc)
	if !ok {
		goto notFound
	}

	// ln.PC <= pcを満たす最後の要素を返す
	for i, ln := range s.data.Lines {
		if ln.PC < mod.MinPC {
			continue
		}
		if ln.PC > mod.MaxPC {
			break
		}
		if ln.PC <= pc {
			found = true
			idx = i
		} else {
			break
		}
	}

	if found {
		return s.data.Lines[idx], true
	}
notFound:
	return GoLine{}, false
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

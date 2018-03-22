package types

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

type SymbolsReadFn func() (SymbolsData, error)
type SymbolsWriteFn func(data SymbolsData) error

type Symbols struct {
	Writable bool

	lock sync.RWMutex
	data SymbolsData
}

// SymbolsData is an immutable symbol table that golang runtime has.
// You MUST NOT modify any fields on SymbolsData except when creating a new object.
type SymbolsData struct {
	Files []string   `json:"files"`
	Mods  []GoModule `json:"modules"`
	Funcs []GoFunc   `json:"functions"`
	Lines []GoLine   `json:"lines"`
}

// FileID is index of Symbols.Files array.
type FileID uint32

// File is file path to the source code.
// example: "/go/src/github.com/yuuki0xff/goapptrace/goapptrace.go"
type File string

// GoModules means a module in golang.
type GoModule struct {
	// Name は空になる場合がある。
	// go1.10の時点での多くのプログラムは、GoModuleはプログラム内にただ一つであり、Nameフィールドは空である。
	Name  string  `json:"name"`
	MinPC uintptr `json:"min-pc"`
	MaxPC uintptr `json:"max-pc"`
}

// GoFunc means a function in golang.
type GoFunc struct {
	// entry point of this function
	Entry uintptr `json:"entry"`
	// example: "github.com/yuuki0xff/goapptrace.main"
	Name string `json:"name"`
}

// GoLine haves a correspondence to position on source code from PC (Program Counter).
type GoLine struct {
	PC uintptr `json:"pc"`
	// file location that defines this function.
	FileID FileID `json:"file-id"`
	Line   uint32 `json:"line"`
}

// 初期化する。使用前に必ず呼び出すこと。
func (s *Symbols) Init() {}

// 指定した状態で初期化する。
// Init()を呼び出す必要はない。
func (s *Symbols) Load(data SymbolsData) {
	if err := data.Validate(); err != nil {
		log.Panic(errors.Wrap(err, "invalid SymbolsData"))
	}
	s.lock.Lock()
	s.data = data
	s.lock.Unlock()
}

// 現在保持している SymbolsData をコールバック関数に渡す。
// fnの内部でファイルへの書き出しなどの処理を行うこと。
// 渡された SymbolsData は変更してはならない。
func (s *Symbols) Save(fn SymbolsWriteFn) error {
	s.lock.RLock()
	data := s.data
	s.lock.RUnlock()
	return fn(data)
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

// File は、pcに対応するファイル名を文字列として返す。
func (s *Symbols) File(pc uintptr) string {
	filename := "?"
	if line, ok := s.GoLine(pc); ok {
		if int(line.FileID) <= len(s.data.Files) {
			filename = s.data.Files[line.FileID]
		}
	}
	return filename
}

// Line は、pcに対応するソースコード上の行番号を返す。
// 行番号が不明の場合、-1を返す。
func (s *Symbols) Line(pc uintptr) int64 {
	linenumber := int64(-1)
	if line, ok := s.GoLine(pc); ok {
		linenumber = int64(line.Line)
	}
	return linenumber
}

// FileLineは、pcに対応するファイル名と行番号を文字列として返す。
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

func (sd *SymbolsData) Validate() error {
	for i, f := range sd.Files {
		if f == "" {
			return fmt.Errorf("sd.Files[%d] is empty", i)
		}
	}

	var pc uintptr
	for i, m := range sd.Mods {
		err := m.Validate()
		if err != nil {
			return err
		}
		if m.MinPC <= pc || m.MaxPC <= pc {
			return fmt.Errorf("sd.Mods is not sorted (i=%d)", i)
		}
		pc = m.MaxPC
	}

	pc = 0
	for i, f := range sd.Funcs {
		err := f.Validate()
		if err != nil {
			return err
		}
		if f.Entry <= pc {
			return fmt.Errorf("sd.Funcs is not sorted (i=%d)", i)
		}
		pc = f.Entry
	}

	pc = 0
	for i, l := range sd.Lines {
		err := l.Validate()
		if err != nil {
			return err
		}
		if l.PC <= pc {
			return fmt.Errorf("sd.Lines is not sorted (i=%d)", i)
		}
		if len(sd.Files) <= int(l.FileID) {
			return fmt.Errorf("sd.Lines[%d].FileID is invalid", i)
		}
		pc = l.PC
	}
	return nil
}
func (m *GoModule) Validate() error {
	if m.MinPC == 0 {
		return errors.New("m.MinPC is 0")
	}
	if m.MaxPC == 0 {
		return errors.New("m.MaxPC is 0")
	}
	if m.MinPC >= m.MaxPC {
		return errors.New("m.MinPC >= m.MaxPC")
	}
	return nil
}

// ShortName strips package path from GoFunc.Name
func (f *GoFunc) ShortName() string {
	// f.Name: example.com/foo/bar.funcname.innerfunc
	// tmp1: bar.funcname.innerfunc
	// tmp2: funcname.innerfunc
	tmp1 := f.Name[strings.LastIndexByte(f.Name, '/')+1:]
	tmp2 := tmp1[strings.IndexByte(tmp1, '.')+1:]
	return tmp2
}

// PackagePath strips function name from GoFunc.Name
func (f *GoFunc) PackagePath() string {
	// f.Name: example.com/foo/bar.funcname.innerfunc
	// tmp1: bar.funcname.innerfunc
	// tmp2: example.com/foo/bar
	p1 := strings.LastIndexByte(f.Name, '/') + 1
	tmp1 := f.Name[p1:]
	p2 := strings.IndexByte(tmp1, '.')
	tmp2 := f.Name[:p1+p2]
	return tmp2
}
func (f *GoFunc) Validate() error {
	if f.Name == "" {
		return errors.New("f.Name is empty")
	}
	if f.Entry == 0 {
		return errors.New("f.Entry is 0")
	}
	return nil
}
func (l *GoLine) Validate() error {
	if l.PC == 0 {
		return errors.New("l.PC is 0")
	}
	if l.Line == 0 {
		return errors.New("l.Line is 0")
	}
	return nil
}

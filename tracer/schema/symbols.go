package schema

import "sync"

type FuncID uint64
type GoLineID uint64

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

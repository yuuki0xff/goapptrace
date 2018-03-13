package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSymbols_ModuleName(t *testing.T) {
	moduleName := "github.com/yuuki0xff/goapptrace/tracer/logutil"
	funcName := moduleName + ".TestSymbols_ModuleName"

	s := Symbols{}
	s.Load(SymbolsData{
		Files: []string{},
		Mods: []GoModule{
			{Name: moduleName, MinPC: 100, MaxPC: 200},
		},
		Funcs: []GoFunc{
			{Entry: 100, Name: funcName},
		},
		Lines: []GoLine{},
	})
	name := s.ModuleName(100)
	if name != moduleName {
		t.Logf("files")
	}
}

func TestSymbols_Func(t *testing.T) {
	a := assert.New(t)
	s := Symbols{}
	s.Load(SymbolsData{
		Mods: []GoModule{
			{Name: "fmt", MinPC: 100, MaxPC: 199},
			{Name: "main", MinPC: 200, MaxPC: 499},
		},
		Funcs: []GoFunc{
			{Entry: 100, Name: "fmt.Sprintf"},
			{Entry: 200, Name: "main.main"},
			{Entry: 300, Name: "main.foo"},
			{Entry: 400, Name: "main.bar"},
		},
	})

	// どのモジュールにもマッチしない場合、関数は常に見つからない。
	_, ok := s.GoFunc(99)
	a.False(ok)
	_, ok = s.GoFunc(500)
	a.False(ok)

	// Funcs配列の最初の関数にマッチするか
	fn, ok := s.GoFunc(100)
	a.True(ok)
	a.Equal("fmt.Sprintf", fn.Name)
	fn, ok = s.GoFunc(110)
	a.True(ok)
	a.Equal("fmt.Sprintf", fn.Name)
	fn, ok = s.GoFunc(199)
	a.True(ok)
	a.Equal("fmt.Sprintf", fn.Name)

	// モジュール内の最初の関数にマッチするか
	fn, ok = s.GoFunc(200)
	a.True(ok)
	a.Equal("main.main", fn.Name)
	fn, ok = s.GoFunc(299)
	a.True(ok)
	a.Equal("main.main", fn.Name)

	// モジュール内の途中の関数にマッチするか
	fn, ok = s.GoFunc(300)
	a.True(ok)
	a.Equal("main.foo", fn.Name)
	_, ok = s.GoFunc(399)
	a.True(ok)
	a.Equal("main.foo", fn.Name)

	// モジュール内の最後の関数にマッチするか
	fn, ok = s.GoFunc(400)
	a.True(ok)
	a.Equal("main.bar", fn.Name)
	fn, ok = s.GoFunc(499)
	a.True(ok)
	a.Equal("main.bar", fn.Name)
}

func TestSymbols_GoLine(t *testing.T) {
	a := assert.New(t)
	s := Symbols{}
	s.Load(SymbolsData{
		Mods: []GoModule{
			{Name: "fmt", MinPC: 100, MaxPC: 199},
			{Name: "main", MinPC: 300, MaxPC: 399},
			{Name: "net", MinPC: 400, MaxPC: 999},
		},
		Lines: []GoLine{
			{PC: 400, FileID: 0, Line: 200},
			{PC: 300, FileID: 0, Line: 100},
			{PC: 310, FileID: 0, Line: 101},
			{PC: 400, FileID: 0, Line: 300},
		},
	})

	// 存在しないモジュールにマッチしない
	_, ok := s.GoLine(0)
	a.False(ok)
	_, ok = s.GoLine(99)
	a.False(ok)
	_, ok = s.GoLine(200)
	a.False(ok)
	_, ok = s.GoLine(299)
	a.False(ok)

	// 最初の行にマッチする
	ln, ok := s.GoLine(300)
	a.True(ok)
	a.Equal(100, ln.Line)
	ln, ok = s.GoLine(309)
	a.True(ok)
	a.Equal(100, ln.Line)

	// 最後の行にマッチする
	ln, ok = s.GoLine(310)
	a.True(ok)
	a.Equal(101, ln.Line)
	ln, ok = s.GoLine(319)
	a.True(ok)
	a.Equal(101, ln.Line)
}

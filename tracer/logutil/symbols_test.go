package logutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSymbols_ModuleName(t *testing.T) {
	moduleName := "github.com/yuuki0xff/goapptrace/tracer/logutil"
	funcName := moduleName + ".TestSymbols_ModuleName"
	funcID := FuncID(0)
	funcSID := FuncStatusID(0)

	sym := Symbols{
		funcs: []*GoFunc{
			{
				Name: funcName,
				ID:   funcID,
			},
		},
		funcStatus: []*FuncStatus{
			{
				ID:   funcSID,
				Func: funcID,
			},
		},
	}
	name := sym.ModuleName(funcSID)
	if name != moduleName {
		t.Logf("files")
	}
}

func TestSymbols_AddFunc_readOnly(t *testing.T) {
	a := assert.New(t)
	s := Symbols{
		Writable: false,
	}
	s.Init()
	a.Panics(func() {
		s.AddFunc(&GoFunc{
			Name:  "test",
			File:  "test.go",
			Entry: 100,
		})
	}, "Symbols is not writable")
}

func TestSymbols_AddFunc_simple(t *testing.T) {
	a := assert.New(t)
	s := Symbols{
		Writable: true,
	}
	s.Init()
	id, added := s.AddFunc(&GoFunc{
		ID:    10,
		Name:  "main.test",
		File:  "test.go",
		Entry: 100,
	})
	a.True(added)
	a.Equal(FuncID(0), id, "First function id is 0. Should not keep original function id.")
	a.Len(s.funcs, 1)

	id, added = s.AddFunc(&GoFunc{
		Name:  "main.test2",
		File:  "test2.go",
		Entry: 200,
	})
	a.True(added)
	a.Equal(FuncID(1), id)
	a.Len(s.funcs, 2)
}
func TestSymbols_AddFunc_dedupRecords(t *testing.T) {
	a := assert.New(t)
	s := Symbols{
		Writable: true,
	}
	s.Init()
	fs := &GoFunc{
		Name:  "main.test2",
		File:  "test2.go",
		Entry: 200,
	}
	id, added := s.AddFunc(fs)
	a.True(added)
	a.Equal(FuncID(0), id)
	a.Len(s.funcs, 1)

	id, added = s.AddFunc(fs)
	a.Equal(false, added)
	a.Equal(FuncID(0), id)
	a.Len(s.funcs, 1)

	// 関数名が一致すれば、その他のフィールドが異なっていても問題ない
	id, added = s.AddFunc(&GoFunc{
		Name: "main.test2",
	})
	a.Equal(false, added)
	a.Equal(FuncID(0), id)
}

func TestSymbols_AddFunc_keepID(t *testing.T) {
	a := assert.New(t)
	s := Symbols{
		Writable: true,
		KeepID:   true,
	}
	s.Init()
	id, added := s.AddFunc(&GoFunc{
		ID:    10,
		Name:  "main.test2",
		File:  "test2.go",
		Entry: 200,
	})
	a.True(added)
	a.Equal(FuncID(10), id)
}

func TestSymbols_AddFuncStatus_simple(t *testing.T) {
	a := assert.New(t)
	s := Symbols{
		Writable: true,
	}
	s.Init()
	id, added := s.AddFuncStatus(&FuncStatus{
		Func: 10, // dummy
		Line: 100,
		PC:   101,
	})
	a.True(added)
	a.Equal(FuncStatusID(0), id)
	a.Len(s.funcStatus, 1)

	id, added = s.AddFuncStatus(&FuncStatus{
		Func: 22, // dummy
		Line: 200,
		PC:   201,
	})
	a.True(added)
	a.Equal(FuncStatusID(1), id)
	a.Len(s.funcStatus, 2)
}

func TestSymbols_AddFuncStatus_keepID(t *testing.T) {
	a := assert.New(t)
	s := Symbols{
		Writable: true,
		KeepID:   true,
	}
	s.Init()
	id, added := s.AddFuncStatus(&FuncStatus{
		ID:   1000,
		Func: 10, // dummy
		Line: 100,
		PC:   101,
	})
	a.True(added)
	a.Equal(FuncStatusID(1000), id)

	id, added = s.AddFuncStatus(&FuncStatus{
		ID:   2200,
		Func: 22, // dummy
		Line: 200,
		PC:   201,
	})
	a.True(added)
	a.Equal(FuncStatusID(2200), id)
}

func TestSymbols_AddFuncStatus_dedup(t *testing.T) {
	a := assert.New(t)
	s := Symbols{
		Writable: true,
		KeepID:   true,
	}
	s.Init()
	rec := &FuncStatus{
		ID:   2,
		Func: 10, // dummy
		Line: 100,
		PC:   101,
	}
	id, added := s.AddFuncStatus(rec)
	a.True(added)
	a.Equal(FuncStatusID(2), id)

	id, added = s.AddFuncStatus(rec)
	a.Equal(false, added)
	a.Equal(FuncStatusID(2), id)

	// PCが一致していれば、他のフィールドの値が異なっていても一致として判定する。
	id, added = s.AddFuncStatus(&FuncStatus{
		PC: 101,
	})
	a.Equal(false, added)
	a.Equal(FuncStatusID(2), id)
}

func TestSymbols_Func(t *testing.T) {
	a := assert.New(t)
	s := Symbols{
		Writable: true,
		KeepID:   true,
	}
	s.Init()

	// 存在しない関数は取得できない
	_, ok := s.Func(FuncID(10))
	a.Equal(false, ok)

	id1, added := s.AddFunc(&GoFunc{
		ID:    10,
		Name:  "main.test2",
		File:  "test2.go",
		Entry: 200,
	})
	a.True(added)
	a.Equal(FuncID(10), id1)

	// 存在するものは取得できる
	f, ok := s.Func(id1)
	a.True(ok)
	a.Equal(id1, f.ID)
}

func TestSymbols_FuncStatus(t *testing.T) {
	a := assert.New(t)
	s := Symbols{
		Writable: true,
		KeepID:   true,
	}
	s.Init()

	_, ok := s.FuncStatus(FuncStatusID(1000))
	a.Equal(false, ok)

	id1, ok := s.AddFuncStatus(&FuncStatus{
		ID:   1000,
		Func: 10, // dummy
		Line: 100,
		PC:   101,
	})
	a.True(ok)
	a.Equal(FuncStatusID(1000), id1)

	fs, ok := s.FuncStatus(id1)
	a.True(ok)
	a.Equal(id1, fs.ID)
}

func TestSymbols_FuncIDFromName(t *testing.T) {
	a := assert.New(t)
	s := Symbols{}
	s.Load(SymbolsData{
		Funcs: []*GoFunc{
			{
				ID:    0,
				Name:  "main.test",
				File:  "main.go",
				Entry: 1000,
			},
		},
		FuncStatus: []*FuncStatus{},
	})

	id, ok := s.FuncIDFromName("main.test")
	a.True(ok)
	a.Equal(FuncID(0), id)

	_, ok = s.FuncIDFromName("not-found")
	a.Equal(false, ok)
}

func TestSymbols_FuncStatusIDFromPC(t *testing.T) {
	a := assert.New(t)
	s := Symbols{}
	s.Load(SymbolsData{
		Funcs: []*GoFunc{
			{
				ID:    0,
				Name:  "main.test",
				File:  "main.go",
				Entry: 1000,
			},
		},
		FuncStatus: []*FuncStatus{
			{
				ID:   0,
				Func: FuncID(0),
				Line: 30,
				PC:   1030,
			},
		},
	})

	id, ok := s.FuncStatusIDFromPC(1030)
	a.True(ok)
	a.Equal(FuncStatusID(0), id)

	// FuncSymbolのEntry pointの値は検索対象外
	_, ok = s.FuncStatusIDFromPC(1000)
	a.Equal(false, ok)

	// 関数の範囲ぽくても、未登録ならfalseを返す。
	_, ok = s.FuncStatusIDFromPC(1010)
	a.Equal(false, ok)
}

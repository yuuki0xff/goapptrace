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
		funcs: []*FuncSymbol{
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
		s.AddFunc(&FuncSymbol{
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
	id, added := s.AddFunc(&FuncSymbol{
		ID:    10,
		Name:  "main.test",
		File:  "test.go",
		Entry: 100,
	})
	a.Equal(true, added)
	a.Equal(FuncID(0), id, "First function id is 0. Should not keep original function id.")
	a.Len(s.funcs, 1)

	id, added = s.AddFunc(&FuncSymbol{
		Name:  "main.test2",
		File:  "test2.go",
		Entry: 200,
	})
	a.Equal(true, added)
	a.Equal(FuncID(1), id)
	a.Len(s.funcs, 2)
}
func TestSymbols_AddFunc_dedupRecords(t *testing.T) {
	a := assert.New(t)
	s := Symbols{
		Writable: true,
	}
	s.Init()
	fs := &FuncSymbol{
		Name:  "main.test2",
		File:  "test2.go",
		Entry: 200,
	}
	id, added := s.AddFunc(fs)
	a.Equal(true, added)
	a.Equal(FuncID(0), id)
	a.Len(s.funcs, 1)

	id, added = s.AddFunc(fs)
	a.Equal(false, added)
	a.Equal(FuncID(0), id)
	a.Len(s.funcs, 1)

	// 関数名が一致すれば、その他のフィールドが異なっていても問題ない
	id, added = s.AddFunc(&FuncSymbol{
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
	id, added := s.AddFunc(&FuncSymbol{
		ID:    10,
		Name:  "main.test2",
		File:  "test2.go",
		Entry: 200,
	})
	a.Equal(true, added)
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
	a.Equal(true, added)
	a.Equal(FuncStatusID(0), id)
	a.Len(s.funcStatus, 1)

	id, added = s.AddFuncStatus(&FuncStatus{
		Func: 22, // dummy
		Line: 200,
		PC:   201,
	})
	a.Equal(true, added)
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
	a.Equal(true, added)
	a.Equal(FuncStatusID(1000), id)

	id, added = s.AddFuncStatus(&FuncStatus{
		ID:   2200,
		Func: 22, // dummy
		Line: 200,
		PC:   201,
	})
	a.Equal(true, added)
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
	a.Equal(true, added)
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

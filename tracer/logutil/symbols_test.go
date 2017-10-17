package logutil

import "testing"

func TestSymbols_ModuleName(t *testing.T) {
	moduleName := "github.com/yuuki0xff/goapptrace/tracer/logutil"
	funcName := moduleName + ".TestSymbols_ModuleName"
	funcID := FuncID(0)
	funcSID := FuncStatusID(0)

	sym := Symbols{
		Funcs: []*FuncSymbol{
			{
				Name: funcName,
				ID:   funcID,
			},
		},
		FuncStatus: []*FuncStatus{
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

func TestSymbolResolver_Init(t *testing.T) {
	dummyFID := FuncID(9999)
	dummyFSID := FuncStatusID(1111)
	f1 := FuncSymbol{
		ID:   dummyFID,
		Name: "example.com/foo/bar.testFunc1",
	}
	f2 := FuncSymbol{
		ID:   dummyFID,
		Name: "example.jp/hoge/mage.testFunc2",
	}
	fs1 := FuncStatus{
		ID: dummyFSID,
	}
	fs2 := FuncStatus{
		ID: dummyFSID,
	}

	sym := Symbols{}
	resolver := SymbolResolver{}
	resolver.Init(&sym)
	fs1.Func, _ = resolver.AddFunc(&f1)
	if f1.ID != FuncID(0) || f1.ID != fs1.Func {
		// FuncSymbol.IDが更新されていない OR 正しいIDを返していない
		t.Errorf("mismatch FuncID: expect 0, actual %d and %d", f1.ID, fs1.Func)
	}

	fsid1, _ := resolver.AddFuncStatus(&fs1)
	if fs1.ID != FuncStatusID(0) || fs1.ID != fsid1 {
		// FuncStatus.IDが更新されていない OR 正しいIDを返していない
		t.Errorf("mismatch FuncStatusID: expect 0, actual %d and %d", fs1.ID, fsid1)
	}

	fs2.Func, _ = resolver.AddFunc(&f2)
	if f2.ID != FuncID(1) {
		t.Errorf("mismatch FuncID: expect 0, actual %d", f2.ID)
	}

	resolver.AddFuncStatus(&fs2)
	if fs2.ID != FuncStatusID(1) {
		t.Errorf("mismatch FuncStatusID: expect 0, actual %d", fs2.ID)
	}
}

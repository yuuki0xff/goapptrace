package logutil

import (
	"log"
	"testing"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func testLoadFromIteratorHelper(t *testing.T, loader *RawLogLoader, symbols *Symbols, testData []RawFuncLogNew) {
	if loader == nil {
		loader = &RawLogLoader{
			RawLogHandler:  func(raw *RawFuncLogNew) {},
			FuncLogHandler: func(flog *FuncLog) {},
		}
	}
	loader.Init()
	loader.SymbolResolver.AddSymbols(symbols)
	var i int
	must(loader.LoadFromIterator(func() (RawFuncLogNew, bool) {
		if i < len(testData) {
			defer func() {
				i++
			}()
			log.Println(i)
			return testData[i], true
		}
		return RawFuncLogNew{}, false
	}))
}

func TestRawLogLoader_LoadFromIterator_startStopFuncs(t *testing.T) {
	txids := []TxID{
		NewTxID(),
	}
	symbols := &Symbols{
		Funcs: []*FuncSymbol{
			{ID: 0, Name: "dummy.main"},
		},
		FuncStatus: []*FuncStatus{
			{ID: 0, Func: 0},
		},
	}
	testData := []RawFuncLogNew{
		// main() start
		{
			Time:      Time(1),
			Tag:       "funcStart",
			Timestamp: 1,
			Frames: []FuncStatusID{
				FuncStatusID(0),
			},
			GID:  0,
			TxID: txids[0],
		},
		// main() end
		{
			Time:      Time(2),
			Tag:       "funcEnd",
			Timestamp: 2,
			Frames: []FuncStatusID{
				FuncStatusID(0),
			},
			GID:  0,
			TxID: txids[0],
		},
	}

	testLoadFromIteratorHelper(t, nil, symbols, testData)
}

func TestRawLogLoader_LoadFromIterator_withNestedCall(t *testing.T) {
	txids := []TxID{
		NewTxID(),
		NewTxID(),
		NewTxID(),
	}
	symbols := &Symbols{
		Funcs: []*FuncSymbol{
			{ID: 0, Name: "dummy.main"},
			{ID: 1, Name: "dummy.func1"},
			{ID: 2, Name: "dummy.func2"},
			{ID: 3, Name: "dummy.newGoroutine"},
		},
		FuncStatus: []*FuncStatus{
			{ID: 0, Func: 0},
			{ID: 1, Func: 1},
			{ID: 2, Func: 2},
			{ID: 3, Func: 3},
		},
	}
	testData := []RawFuncLogNew{
		// main() start
		{
			Time:      Time(1),
			Tag:       "funcStart",
			Timestamp: 1,
			Frames: []FuncStatusID{
				FuncStatusID(0),
			},
			GID:  0,
			TxID: txids[0],
		},
		// main() -> func1() start
		{
			Time:      Time(2),
			Tag:       "funcStart",
			Timestamp: 2,
			Frames: []FuncStatusID{
				FuncStatusID(0),
				FuncStatusID(1),
			},
			GID:  0,
			TxID: txids[1],
		},
		// main() -> func1() -> func2() start
		{
			Time:      Time(3),
			Tag:       "funcStart",
			Timestamp: 3,
			Frames: []FuncStatusID{
				FuncStatusID(0),
				FuncStatusID(1),
				FuncStatusID(2),
			},
			GID:  0,
			TxID: txids[2],
		},
		// main() -> func1() -> func2() end
		{
			Time:      Time(4),
			Tag:       "funcEnd",
			Timestamp: 4,
			Frames: []FuncStatusID{
				FuncStatusID(0),
				FuncStatusID(1),
				FuncStatusID(2),
			},
			GID:  0,
			TxID: txids[2],
		},
		// main() -> func1() end
		{
			Time:      Time(5),
			Tag:       "funcEnd",
			Timestamp: 5,
			Frames: []FuncStatusID{
				FuncStatusID(0),
				FuncStatusID(1),
			},
			GID:  0,
			TxID: txids[1],
		},
		// main() end
		{
			Time:      Time(6),
			Tag:       "funcEnd",
			Timestamp: 6,
			Frames: []FuncStatusID{
				FuncStatusID(0),
			},
			GID:  0,
			TxID: txids[0],
		},
	}

	testLoadFromIteratorHelper(t, nil, symbols, testData)
}

func TestRawLogLoader_LoadFromIterator_startStopNewGoroutines(t *testing.T) {
	txids := []TxID{
		NewTxID(),
		NewTxID(),
	}
	symbols := &Symbols{
		Funcs: []*FuncSymbol{
			{ID: 0, Name: "dummy.main"},
			{ID: 0, Name: "dummy.newGoroutine"},
		},
		FuncStatus: []*FuncStatus{
			{ID: 0, Func: 0},
			{ID: 1, Func: 1},
		},
	}
	testData := []RawFuncLogNew{
		// main() start
		{
			Time:      Time(1),
			Tag:       "funcStart",
			Timestamp: 1,
			Frames: []FuncStatusID{
				FuncStatusID(0),
			},
			GID:  0,
			TxID: txids[0],
		},
		// main()
		// newGoroutine() start
		{
			Time:      Time(2),
			Tag:       "funcStart",
			Timestamp: 2,
			Frames: []FuncStatusID{
				FuncStatusID(1),
			},
			GID:  1,
			TxID: txids[1],
		},
		// main()
		// newGoroutine() end
		{
			Time:      Time(3),
			Tag:       "funcEnd",
			Timestamp: 3,
			Frames: []FuncStatusID{
				FuncStatusID(1),
			},
			GID:  1,
			TxID: txids[1],
		},
		// main() end
		{
			Time:      Time(4),
			Tag:       "funcEnd",
			Timestamp: 4,
			Frames: []FuncStatusID{
				FuncStatusID(0),
			},
			GID:  0,
			TxID: txids[0],
		},
	}

	testLoadFromIteratorHelper(t, nil, symbols, testData)
}

func TestRawLogLoader_LoadFromIterator_handlerIsNil(t *testing.T) {
	txids := []TxID{
		NewTxID(),
	}
	symbols := &Symbols{
		Funcs: []*FuncSymbol{
			{ID: 0, Name: "dummy.main"},
		},
		FuncStatus: []*FuncStatus{
			{ID: 0, Func: 0},
		},
	}
	testData := []RawFuncLogNew{
		// main() start
		{
			Time:      Time(1),
			Tag:       "funcStart",
			Timestamp: 1,
			Frames: []FuncStatusID{
				FuncStatusID(0),
			},
			GID:  0,
			TxID: txids[0],
		},
		// main() end
		{
			Time:      Time(2),
			Tag:       "funcEnd",
			Timestamp: 2,
			Frames: []FuncStatusID{
				FuncStatusID(0),
			},
			GID:  0,
			TxID: txids[0],
		},
	}

	testLoadFromIteratorHelper(t, &RawLogLoader{
		RawLogHandler:  nil,
		FuncLogHandler: nil,
	}, symbols, testData)
}

func TestRawLogLoader_LoadFromIterator_endlessFuncs(t *testing.T) {
	txids := []TxID{
		NewTxID(),
	}
	symbols := &Symbols{
		Funcs: []*FuncSymbol{
			{ID: 0, Name: "dummy.main"},
		},
		FuncStatus: []*FuncStatus{
			{ID: 0, Func: 0},
		},
	}
	testData := []RawFuncLogNew{
		// main() start
		{
			Time:      Time(1),
			Tag:       "funcStart",
			Timestamp: 1,
			Frames: []FuncStatusID{
				FuncStatusID(0),
			},
			GID:  0,
			TxID: txids[0],
		},
	}
	testLoadFromIteratorHelper(t, nil, symbols, testData)
}

func TestRawLogLoader_LoadFromJsonLines(t *testing.T) {
	loader := RawLogLoader{
		RawLogHandler:  func(raw *RawFuncLogNew) {},
		FuncLogHandler: func(flog *FuncLog) {},
	}
	loader.Init()
	// TODO: LoadFromJsonLinesをテストする。
	//must(loader.LoadFromJsonLines(nil))
}

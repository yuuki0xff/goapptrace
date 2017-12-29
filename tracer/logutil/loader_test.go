package logutil

import (
	"testing"
)

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func testStateSimulatorHelper(t *testing.T, s *StateSimulator, symbols *Symbols, testData []RawFuncLog) {
	if s == nil {
		s = &StateSimulator{}
	}
	s.Init()
	for _, data := range testData {
		s.Next(data)
	}
}

func TestStateSimulator_Next_startStopFuncs(t *testing.T) {
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
	testData := []RawFuncLog{
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

	testStateSimulatorHelper(t, nil, symbols, testData)
}

func TestStateSimulator_Next_withNestedCall(t *testing.T) {
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
	testData := []RawFuncLog{
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

	testStateSimulatorHelper(t, nil, symbols, testData)
}

func TestStateSimulator_Next_startStopNewGoroutines(t *testing.T) {
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
	testData := []RawFuncLog{
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

	testStateSimulatorHelper(t, nil, symbols, testData)
}

func TestStateSimulator_Next_handlerIsNil(t *testing.T) {
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
	testData := []RawFuncLog{
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

	testStateSimulatorHelper(t, &StateSimulator{}, symbols, testData)
}

func TestStateSimulator_Next_endlessFuncs(t *testing.T) {
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
	testData := []RawFuncLog{
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
	testStateSimulatorHelper(t, nil, symbols, testData)
}

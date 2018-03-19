package simulator

import (
	"testing"

	"github.com/yuuki0xff/goapptrace/tracer/types"
)

func testStateSimulatorHelper(t *testing.T, s *StateSimulator, symbols *types.Symbols, testData []types.RawFuncLog) {
	if s == nil {
		s = &StateSimulator{}
	}
	s.Init()
	for _, data := range testData {
		s.Next(data)
	}
}

func TestStateSimulator_Next_startStopFuncs(t *testing.T) {
	txids := []types.TxID{
		types.NewTxID(),
	}
	symbols := &types.Symbols{}
	symbols.Load(types.SymbolsData{
		Mods: []types.GoModule{
			{Name: "main", MinPC: 100, MaxPC: 500},
		},
		Funcs: []types.GoFunc{
			{Entry: 200, Name: "main.main"},
		},
	})
	testData := []types.RawFuncLog{
		// main() start
		{
			ID:        types.RawFuncLogID(1),
			Tag:       types.FuncStart,
			Timestamp: 1,
			Frames:    []uintptr{101},
			GID:       0,
			TxID:      txids[0],
		},
		// main() end
		{
			ID:        types.RawFuncLogID(2),
			Tag:       types.FuncEnd,
			Timestamp: 2,
			Frames:    []uintptr{110},
			GID:       0,
			TxID:      txids[0],
		},
	}

	testStateSimulatorHelper(t, nil, symbols, testData)
}

func TestStateSimulator_Next_withNestedCall(t *testing.T) {
	txids := []types.TxID{
		types.NewTxID(),
		types.NewTxID(),
		types.NewTxID(),
	}
	symbols := &types.Symbols{}
	symbols.Load(types.SymbolsData{
		Mods: []types.GoModule{
			{Name: "main", MinPC: 100, MaxPC: 999},
		},
		Funcs: []types.GoFunc{
			{Entry: 100, Name: "main.main"},
			{Entry: 200, Name: "main.func1"},
			{Entry: 300, Name: "main.func2"},
		},
	})
	testData := []types.RawFuncLog{
		// main() start
		{
			ID:        types.RawFuncLogID(1),
			Tag:       types.FuncStart,
			Timestamp: 1,
			Frames:    []uintptr{100},
			GID:       0,
			TxID:      txids[0],
		},
		// main() -> func1() start
		{
			ID:        types.RawFuncLogID(2),
			Tag:       types.FuncStart,
			Timestamp: 2,
			Frames:    []uintptr{110, 200},
			GID:       0,
			TxID:      txids[1],
		},
		// main() -> func1() -> func2() start
		{
			ID:        types.RawFuncLogID(3),
			Tag:       types.FuncStart,
			Timestamp: 3,
			Frames:    []uintptr{110, 210, 300},
			GID:       0,
			TxID:      txids[2],
		},
		// main() -> func1() -> func2() end
		{
			ID:        types.RawFuncLogID(4),
			Tag:       types.FuncEnd,
			Timestamp: 4,
			Frames:    []uintptr{110, 210, 320},
			GID:       0,
			TxID:      txids[2],
		},
		// main() -> func1() end
		{
			ID:        types.RawFuncLogID(5),
			Tag:       types.FuncEnd,
			Timestamp: 5,
			Frames:    []uintptr{110, 220},
			GID:       0,
			TxID:      txids[1],
		},
		// main() end
		{
			ID:        types.RawFuncLogID(6),
			Tag:       types.FuncEnd,
			Timestamp: 6,
			Frames:    []uintptr{120},
			GID:       0,
			TxID:      txids[0],
		},
	}

	testStateSimulatorHelper(t, nil, symbols, testData)
}

func TestStateSimulator_Next_startStopNewGoroutines(t *testing.T) {
	txids := []types.TxID{
		types.NewTxID(),
		types.NewTxID(),
	}
	symbols := &types.Symbols{}
	symbols.Load(types.SymbolsData{
		Mods: []types.GoModule{
			{Name: "main", MinPC: 100, MaxPC: 999},
		},
		Funcs: []types.GoFunc{
			{Entry: 100, Name: "main.main"},
			{Entry: 200, Name: "main.newGoroutine"},
		},
	})
	testData := []types.RawFuncLog{
		// main() start
		{
			ID:        types.RawFuncLogID(1),
			Tag:       types.FuncStart,
			Timestamp: 1,
			Frames:    []uintptr{100},
			GID:       0,
			TxID:      txids[0],
		},
		// main()
		// newGoroutine() start
		{
			ID:        types.RawFuncLogID(2),
			Tag:       types.FuncStart,
			Timestamp: 2,
			Frames:    []uintptr{200},
			GID:       1,
			TxID:      txids[1],
		},
		// main()
		// newGoroutine() end
		{
			ID:        types.RawFuncLogID(3),
			Tag:       types.FuncEnd,
			Timestamp: 3,
			Frames:    []uintptr{210},
			GID:       1,
			TxID:      txids[1],
		},
		// main() end
		{
			ID:        types.RawFuncLogID(4),
			Tag:       types.FuncEnd,
			Timestamp: 4,
			Frames:    []uintptr{110},
			GID:       0,
			TxID:      txids[0],
		},
	}

	testStateSimulatorHelper(t, nil, symbols, testData)
}

// TODO: ?
func TestStateSimulator_Next_handlerIsNil(t *testing.T) {
	txids := []types.TxID{
		types.NewTxID(),
	}
	symbols := &types.Symbols{}
	symbols.Load(types.SymbolsData{
		Mods: []types.GoModule{
			{Name: "main", MinPC: 100, MaxPC: 999},
		},
		Funcs: []types.GoFunc{
			{Entry: 100, Name: "main.main"},
		},
	})
	testData := []types.RawFuncLog{
		// main() start
		{
			ID:        types.RawFuncLogID(1),
			Tag:       types.FuncStart,
			Timestamp: 1,
			Frames:    []uintptr{100},
			GID:       0,
			TxID:      txids[0],
		},
		// main() end
		{
			ID:        types.RawFuncLogID(2),
			Tag:       types.FuncEnd,
			Timestamp: 2,
			Frames:    []uintptr{110},
			GID:       0,
			TxID:      txids[0],
		},
	}

	testStateSimulatorHelper(t, &StateSimulator{}, symbols, testData)
}

func TestStateSimulator_Next_endlessFuncs(t *testing.T) {
	txids := []types.TxID{
		types.NewTxID(),
	}
	symbols := &types.Symbols{}
	symbols.Load(types.SymbolsData{
		Mods: []types.GoModule{
			{Name: "main", MinPC: 100, MaxPC: 999},
		},
		Funcs: []types.GoFunc{
			{Entry: 100, Name: "main.main"},
		},
	})
	testData := []types.RawFuncLog{
		// main() start
		{
			ID:        types.RawFuncLogID(1),
			Tag:       types.FuncStart,
			Timestamp: 1,
			Frames:    []uintptr{100},
			GID:       0,
			TxID:      txids[0],
		},
	}
	testStateSimulatorHelper(t, nil, symbols, testData)
}

package storage

import (
	"os"
	"testing"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

func doTestSymbolsReaderWriter(
	t *testing.T,
	writerFunc func(sw *SymbolsWriter),
	checkFunc func(symbols *logutil.Symbols),
) {
	file := createTempFile()
	defer must(t, os.Remove(string(file)), "Delete tmpfile:")

	// writing phase
	{
		sw := SymbolsWriter{
			File: file,
		}
		must(t, sw.Open(), "SymbolsWriter.Open():")
		writerFunc(&sw)
		//must(t, sw.Append(), "SymbolsWriter.Append():")
		must(t, sw.Close(), "SymbolsWriter.Close():")
	}

	// reading phase
	{
		symbols := &logutil.Symbols{
			Writable: true,
			KeepID:   true,
		}
		symbols.Init()

		sr := SymbolsReader{
			File:    file,
			Symbols: symbols,
		}
		must(t, sr.Open(), "SymbolsReader.Open():")
		must(t, sr.Load(), "SymbolsReader.Load():")
		must(t, sr.Close(), "SymbolsReader.Close():")

		checkFunc(symbols)
	}
}

func TestSymbolsReaderWriter_loadEmptyFile(t *testing.T) {
	doTestSymbolsReaderWriter(
		t,
		// write
		func(sw *SymbolsWriter) {},
		// check data
		func(symbols *logutil.Symbols) {
			if len(symbols.funcStatus) != 0 {
				t.Errorf("Expected FuncStatus is empty, but %+v", symbols.funcStatus)
			}
			if len(symbols.funcs) != 0 {
				t.Errorf("Expected Funcs is empty, but %+v", symbols.funcs)
			}
		},
	)
}

func TestSymbolsReaderWriter_emptySymbols(t *testing.T) {
	newSymbols := func() *logutil.Symbols {
		symbols := &logutil.Symbols{
			Writable: true,
			KeepID:   true,
		}
		symbols.Init()
		return symbols
	}
	doTestSymbolsReaderWriter(
		t,
		// write
		func(sw *SymbolsWriter) {
			must(t, sw.Append(newSymbols()), "SymbolsWriter.Append():")
			must(t, sw.Append(newSymbols()), "SymbolsWriter.Append():")
			must(t, sw.Append(newSymbols()), "SymbolsWriter.Append():")
		},
		// check data
		func(symbols *logutil.Symbols) {
			if len(symbols.funcStatus) != 0 {
				t.Errorf("Expected FuncStatus is empty, but %+v", symbols.funcStatus)
			}
			if len(symbols.funcs) != 0 {
				t.Errorf("Expected Funcs is empty, but %+v", symbols.funcs)
			}
		},
	)
}
func TestSymbolsReaderWrieter_data(t *testing.T) {
	var fIDs [2]logutil.FuncID
	var fsIDs [2]logutil.FuncStatusID
	funcSymbols := []*logutil.FuncSymbol{
		{
			Name:  "github.com/yuuki0xff/dummyModuleName.main",
			File:  "/src/github.com/yuuki0xff/dummyModuleName/main.go",
			Entry: 1,
		}, {
			Name:  "github.com/yuuki0xff/dummyModuleName.OtherFunc",
			File:  "/src/github.com/yuuki0xff/dummyModuleName/util.go",
			Entry: 100,
		},
	}
	funcStatuses := []*logutil.FuncStatus{
		{
			//Func: fIDs[0],
			Line: 10,
			PC:   11,
		}, {
			//Func: fIDs[1],
			Line: 110,
			PC:   111,
		},
	}

	doTestSymbolsReaderWriter(
		t,
		// write
		func(sw *SymbolsWriter) {
			s := &logutil.Symbols{
				Writable: true,
				KeepID:   false,
			}
			s.Init()

			fIDs[0], _ = s.AddFunc(funcSymbols[0])
			funcStatuses[0].Func = fIDs[0]
			fsIDs[0], _ = s.AddFuncStatus(funcStatuses[0])

			fIDs[1], _ = s.AddFunc(funcSymbols[1])
			funcStatuses[1].Func = fIDs[1]
			fsIDs[1], _ = s.AddFuncStatus(funcStatuses[1])
			sw.Append(s)
		},
		// check data
		func(symbols *logutil.Symbols) {
			for i := range symbols.funcs {
				t.Logf("Funcs[%d] = %+v", i, symbols.funcs[i])
			}
			for i := range symbols.funcStatus {
				t.Logf("FuncStatu[%d] = %+v", i, symbols.funcStatus[i])
			}

			if len(symbols.funcs) != 2 {
				t.Errorf("Mismatched length of Funcs array: len(Funcs)=%d != 2", len(symbols.funcs))
			}
			if !(*symbols.funcs[fIDs[0]] == *funcSymbols[0] && *symbols.funcs[fIDs[1]] == *funcSymbols[1]) {
				t.Errorf("Mismatched FuncSymbol object")
			}
			if len(symbols.funcStatus) != 2 {
				t.Errorf("Mismatched length of FuncStatus array: len(FuncStatus)=%d != 2", len(symbols.funcStatus))
			}
			if !(*symbols.funcStatus[fsIDs[0]] == *funcStatuses[0] && *symbols.funcStatus[fsIDs[1]] == *funcStatuses[1]) {
				t.Errorf("Mismatched FuncStatus object")
			}
		},
	)
}

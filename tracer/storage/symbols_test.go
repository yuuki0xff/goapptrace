package storage

import (
	"testing"

	"os"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

func newSymbols() (*logutil.Symbols, *logutil.SymbolResolver) {
	symbols := &logutil.Symbols{}
	symbols.Init()

	sresolve := &logutil.SymbolResolver{}
	sresolve.Init(symbols)
	return symbols, sresolve
}

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
		symbols := &logutil.Symbols{}
		symbols.Init()
		sresolve := &logutil.SymbolResolver{}
		sresolve.Init(symbols)

		sr := SymbolsReader{
			File:           file,
			SymbolResolver: sresolve,
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
			if len(symbols.FuncStatus) != 0 {
				t.Errorf("Expected FuncStatus is empty, but %+v", symbols.FuncStatus)
			}
			if len(symbols.Funcs) != 0 {
				t.Errorf("Expected Funcs is empty, but %+v", symbols.Funcs)
			}
		},
	)
}

func TestSymbolsReaderWriter_emptySymbols(t *testing.T) {
	doTestSymbolsReaderWriter(
		t,
		// write
		func(sw *SymbolsWriter) {
			symbols, _ := newSymbols()
			must(t, sw.Append(symbols), "SymbolsWriter.Append():")

			symbols, _ = newSymbols()
			must(t, sw.Append(symbols), "SymbolsWriter.Append():")

			symbols, _ = newSymbols()
			must(t, sw.Append(symbols), "SymbolsWriter.Append():")
		},
		// check data
		func(symbols *logutil.Symbols) {
			if len(symbols.FuncStatus) != 0 {
				t.Errorf("Expected FuncStatus is empty, but %+v", symbols.FuncStatus)
			}
			if len(symbols.Funcs) != 0 {
				t.Errorf("Expected Funcs is empty, but %+v", symbols.Funcs)
			}
		},
	)
}
func TestSymbolsReaderWrieter_data(t *testing.T) {
	var funcID1, funcID2 logutil.FuncID
	var funcStatusID1, funcStatusID2 logutil.FuncStatusID

	doTestSymbolsReaderWriter(
		t,
		// write
		func(sw *SymbolsWriter) {
			s, sr := newSymbols()
			funcID1, _ = sr.AddFunc(&logutil.FuncSymbol{
				Name:  "github.com/yuuki0xff/dummyModuleName.main",
				File:  "/src/github.com/yuuki0xff/dummyModuleName/main.go",
				Entry: 1,
			})
			funcStatusID1, _ = sr.AddFuncStatus(&logutil.FuncStatus{
				Func: funcID1,
				Line: 10,
				PC:   11,
			})
			funcID2, _ = sr.AddFunc(&logutil.FuncSymbol{
				Name:  "github.com/yuuki0xff/dummyModuleName.OtherFunc",
				File:  "/src/github.com/yuuki0xff/dummyModuleName/util.go",
				Entry: 100,
			})
			funcStatusID2, _ = sr.AddFuncStatus(&logutil.FuncStatus{
				Func: funcID2,
				Line: 110,
				PC:   111,
			})
			sw.Append(s)
		},
		// check data
		func(symbols *logutil.Symbols) {
			for i := range symbols.Funcs {
				t.Logf("Funcs[%d] = %+v", i, symbols.Funcs[i])
			}
			for i := range symbols.FuncStatus {
				t.Logf("FuncStatu[%d] = %+v", i, symbols.FuncStatus[i])
			}

			if len(symbols.Funcs) != 2 {
				t.Errorf("Mismatched length of Funcs array: len(Funcs)=%d != 2", len(symbols.Funcs))
			}
			if len(symbols.FuncStatus) != 2 {
				t.Errorf("Mismatched length of FuncStatus array: len(FuncStatus)=%d != 2", len(symbols.FuncStatus))
			}
		},
	)
}

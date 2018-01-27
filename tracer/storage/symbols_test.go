package storage

import (
	"bytes"
	"fmt"
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
			t.Log(symbols2string(symbols))
			if symbols.FuncsSize() != 0 {
				t.Errorf("Expected Funcs slice is empty, but it has %d elements", symbols.FuncsSize())
			}
			if symbols.FuncStatusSize() != 0 {
				t.Errorf("Expected FuncStatus slice is empty, but it has %d elements", symbols.FuncStatusSize())
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
			t.Log(symbols2string(symbols))
			if symbols.FuncsSize() != 0 {
				t.Errorf("Expected Funcs slice is empty, but it has %d elements", symbols.FuncsSize())
			}
			if symbols.FuncStatusSize() != 0 {
				t.Errorf("Expected FuncStatus slice is empty, but it has %d elements", symbols.FuncStatusSize())
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
			t.Log(symbols2string(symbols))

			if symbols.FuncsSize() != 2 {
				t.Errorf("Mismatched length of Funcs array: len(Funcs)=%d != 2", symbols.FuncsSize())
			}
			f1, _ := symbols.Func(0)
			f2, _ := symbols.Func(1)
			if !(f1 == *funcSymbols[0] && f2 == *funcSymbols[1]) {
				t.Errorf("Mismatched FuncSymbol object")
			}

			if symbols.FuncStatusSize() != 2 {
				t.Errorf("Mismatched length of FuncStatus array: len(FuncStatus)=%d != 2", symbols.FuncStatusSize())
			}
			fs1, _ := symbols.FuncStatus(0)
			fs2, _ := symbols.FuncStatus(1)
			if !(fs1 == *funcStatuses[0] && fs2 == *funcStatuses[1]) {
				t.Errorf("Mismatched FuncStatus object")
			}
		},
	)
}

func symbols2string(symbols *logutil.Symbols) string {
	buf := bytes.NewBuffer(nil)

	fmt.Println(buf, "Symbols.Funcs:")
	symbols.WalkFuncs(func(fs logutil.FuncSymbol) error {
		fmt.Fprintf(buf, "  Funcs[%d] = %+v", fs.ID, fs)
		return nil
	})

	fmt.Println(buf, "Symbols.FuncStatus:")
	symbols.WalkFuncStatus(func(fs logutil.FuncStatus) error {
		fmt.Fprintf(buf, "  FuncStatu[%d] = %+v", fs.ID, fs)
		return nil
	})
	return buf.String()
}

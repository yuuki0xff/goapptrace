package storage

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

func doTestSymbolsReaderWriter(
	t *testing.T,
	writerFunc func(sw *SymbolsWriter),
	checkFunc func(symbols *logutil.Symbols),
) {
	a := assert.New(t)
	file := createTempFile()
	defer a.NoError(os.Remove(string(file)))

	// writing phase
	{
		sw := SymbolsWriter{
			File: file,
		}
		a.NoError(sw.Open())
		writerFunc(&sw)
		//a.NoError(sw.Append())
		a.NoError(sw.Close())
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
		a.NoError(sr.Open())
		a.NoError(sr.Load())
		a.NoError(sr.Close())

		checkFunc(symbols)
	}
}

func TestSymbolsReaderWriter_loadEmptyFile(t *testing.T) {
	a := assert.New(t)
	doTestSymbolsReaderWriter(
		t,
		// write
		func(sw *SymbolsWriter) {},
		// check data
		func(symbols *logutil.Symbols) {
			t.Log(symbols2string(symbols))
			a.Equal(symbols.FuncsSize(), 0)
			a.Equal(symbols.FuncStatusSize(), 0)
		},
	)
}

func TestSymbolsReaderWriter_emptySymbols(t *testing.T) {
	a := assert.New(t)
	doTestSymbolsReaderWriter(
		t,
		// write
		func(sw *SymbolsWriter) {
			a.NoError(sw.Append(&logutil.SymbolsDiff{}))
			a.NoError(sw.Append(&logutil.SymbolsDiff{}))
			a.NoError(sw.Append(&logutil.SymbolsDiff{}))
		},
		// check data
		func(symbols *logutil.Symbols) {
			t.Log(symbols2string(symbols))
			a.Equal(symbols.FuncsSize(), 0)
			a.Equal(symbols.FuncStatusSize(), 0)
		},
	)
}
func TestSymbolsReaderWrieter_data(t *testing.T) {
	a := assert.New(t)
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

			a.NoError(s.Save(func(diff logutil.SymbolsDiff) error {
				return sw.Append(&diff)
			}), "failed to write symbols diff")
		},
		// check data
		func(symbols *logutil.Symbols) {
			t.Log(symbols2string(symbols))

			a.Equal(symbols.FuncsSize(), 2, "Mismatched length of Funcs array")
			f1, _ := symbols.Func(0)
			f2, _ := symbols.Func(1)
			a.Equal(f1, *funcSymbols[0], "Mismatched FuncSymbol object")
			a.Equal(f2, *funcSymbols[1], "Mismatched FuncSymbol object")

			a.Equal(symbols.FuncStatusSize(), 2, "Mismatched length of FuncStatus array")
			fs1, _ := symbols.FuncStatus(0)
			fs2, _ := symbols.FuncStatus(1)
			a.Equal(fs1, *funcStatuses[0], "Mismatched FuncStatus object")
			a.Equal(fs2, *funcStatuses[1], "Mismatched FuncStatus object")
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

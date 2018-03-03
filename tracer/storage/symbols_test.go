package storage

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

func doTestSymbolsStore(
	t *testing.T,
	writerFunc func(symbols *logutil.Symbols),
	checkFunc func(symbols *logutil.Symbols),
) {
	a := assert.New(t)
	file := createTempFile()
	defer a.NoError(os.Remove(string(file)))

	store := SymbolsStore{
		File: file,
	}
	symbols := logutil.Symbols{
		Writable: true,
	}
	symbols.Init()

	// writing phase
	{
		writerFunc(&symbols)
		a.NoError(store.Write(&symbols))
	}

	// reading phase
	{
		store.ReadOnly = true
		a.NoError(store.Read(&symbols))
		checkFunc(&symbols)
	}
}

func TestSymbolsStore_loadEmptyFile(t *testing.T) {
	a := assert.New(t)
	doTestSymbolsStore(
		t,
		// write
		func(symbols *logutil.Symbols) {},
		// check data
		func(symbols *logutil.Symbols) {
			t.Log(symbols2string(symbols))
			a.Equal(0, symbols.FuncsSize())
			a.Equal(0, symbols.FuncStatusSize())
		},
	)
}

func TestSymbolsStore_addASymbol(t *testing.T) {
	a := assert.New(t)
	doTestSymbolsStore(
		t,
		// write
		func(s *logutil.Symbols) {
			s.AddFunc(&logutil.FuncSymbol{})
			s.AddFuncStatus(&logutil.FuncStatus{})
		},
		// check data
		func(symbols *logutil.Symbols) {
			t.Log(symbols2string(symbols))
			a.Equal(1, symbols.FuncsSize())
			a.Equal(1, symbols.FuncStatusSize())
		},
	)
}
func TestSymbolsStore_addSymbolsWithData(t *testing.T) {
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

	doTestSymbolsStore(
		t,
		// write
		func(s *logutil.Symbols) {
			fIDs[0], _ = s.AddFunc(funcSymbols[0])
			funcStatuses[0].Func = fIDs[0]
			fsIDs[0], _ = s.AddFuncStatus(funcStatuses[0])

			fIDs[1], _ = s.AddFunc(funcSymbols[1])
			funcStatuses[1].Func = fIDs[1]
			fsIDs[1], _ = s.AddFuncStatus(funcStatuses[1])
		},
		// check data
		func(symbols *logutil.Symbols) {
			t.Log(symbols2string(symbols))

			a.Equal(2, symbols.FuncsSize(), "Mismatched length of Funcs array")
			f1, _ := symbols.Func(0)
			f2, _ := symbols.Func(1)
			a.Equal(*funcSymbols[0], f1, "Mismatched FuncSymbol object")
			a.Equal(*funcSymbols[1], f2, "Mismatched FuncSymbol object")

			a.Equal(2, symbols.FuncStatusSize(), "Mismatched length of FuncStatus array")
			fs1, _ := symbols.FuncStatus(0)
			fs2, _ := symbols.FuncStatus(1)
			a.Equal(*funcStatuses[0], fs1, "Mismatched FuncStatus object")
			a.Equal(*funcStatuses[1], fs2, "Mismatched FuncStatus object")
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

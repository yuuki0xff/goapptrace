package storage

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/util"
)

func doTestSymbolsStore(
	t *testing.T,
	writerFunc func(symbols *logutil.Symbols),
	checkFunc func(symbols *logutil.Symbols),
) {
	a := assert.New(t)
	util.WithTempFile(func(tmpfile string) {
		file := File(tmpfile)
		defer a.NoError(os.Remove(string(file)))

		store := SymbolsStore{
			File: file,
		}

		// writing phase
		{
			symbols := logutil.Symbols{
				Writable: true,
			}
			symbols.Init()

			writerFunc(&symbols)
			a.NoError(store.Write(&symbols))
		}

		// reading phase
		{
			symbols := logutil.Symbols{
				Writable: true,
			}
			symbols.Init()

			a.NoError(store.Read(&symbols))
			checkFunc(&symbols)
		}
	})
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
			a.Equal(0, symbols.GoLineSize())
		},
	)
}

func TestSymbolsStore_addASymbol(t *testing.T) {
	a := assert.New(t)
	doTestSymbolsStore(
		t,
		// write
		func(s *logutil.Symbols) {
			s.AddFunc(&logutil.GoFunc{})
			s.AddGoLine(&logutil.GoLine{})
		},
		// check data
		func(symbols *logutil.Symbols) {
			t.Log(symbols2string(symbols))
			a.Equal(1, symbols.FuncsSize())
			a.Equal(1, symbols.GoLineSize())
		},
	)
}
func TestSymbolsStore_addSymbolsWithData(t *testing.T) {
	a := assert.New(t)
	var fIDs [2]logutil.FuncID
	var fsIDs [2]logutil.GoLineID
	goFuncs := []*logutil.GoFunc{
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
	goLines := []*logutil.GoLine{
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
			fIDs[0], _ = s.AddFunc(goFuncs[0])
			goLines[0].Func = fIDs[0]
			fsIDs[0], _ = s.AddGoLine(goLines[0])

			fIDs[1], _ = s.AddFunc(goFuncs[1])
			goLines[1].Func = fIDs[1]
			fsIDs[1], _ = s.AddGoLine(goLines[1])
		},
		// check data
		func(symbols *logutil.Symbols) {
			t.Log(symbols2string(symbols))

			a.Equal(2, symbols.FuncsSize(), "Mismatched length of Funcs array")
			f1, _ := symbols.Func(0)
			f2, _ := symbols.Func(1)
			a.Equal(*goFuncs[0], f1, "Mismatched GoFunc object")
			a.Equal(*goFuncs[1], f2, "Mismatched GoFunc object")

			a.Equal(2, symbols.GoLineSize(), "Mismatched length of GoLine array")
			fs1, _ := symbols.GoLine(0)
			fs2, _ := symbols.GoLine(1)
			a.Equal(*goLines[0], fs1, "Mismatched GoLine object")
			a.Equal(*goLines[1], fs2, "Mismatched GoLine object")
		},
	)
}

func symbols2string(symbols *logutil.Symbols) string {
	buf := bytes.NewBuffer(nil)

	fmt.Println(buf, "Symbols.Funcs:")
	symbols.WalkFuncs(func(fs logutil.GoFunc) error {
		fmt.Fprintf(buf, "  Funcs[%d] = %+v", fs.ID, fs)
		return nil
	})

	fmt.Println(buf, "Symbols.Lines:")
	symbols.WalkGoLine(func(fs logutil.GoLine) error {
		fmt.Fprintf(buf, "  GoLine[%d] = %+v", fs.ID, fs)
		return nil
	})
	return buf.String()
}

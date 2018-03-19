package storage

import (
	"bytes"
	"testing"

	"github.com/kr/pretty"
	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/goapptrace/tracer/types"
	"github.com/yuuki0xff/goapptrace/tracer/util"
)

func TestSymbolsStore_Write(t *testing.T) {
	t.Run("read-only", func(t *testing.T) {
		a := assert.New(t)
		util.WithTempFile(func(tmpfile string) {
			ss := SymbolsStore{
				File:     File(tmpfile),
				ReadOnly: true,
			}
			a.EqualError(ss.Write(emptySymbols()), "SymbolsStore: read only")
		})
	})
}
func TestSymbolsStore_ReadWrite(t *testing.T) {
	testReadWrite := func(t *testing.T, expected types.SymbolsData, write *types.Symbols) {
		a := assert.New(t)
		util.WithTempFile(func(tmpfile string) {
			ss := SymbolsStore{
				File: File(tmpfile),
			}
			// write
			a.NoError(ss.Write(emptySymbols()))

			// read
			s := &types.Symbols{}
			a.NoError(ss.Read(s))
			called := false
			s.Save(func(data types.SymbolsData) error {
				called = true
				t.Log(symbolsData2string(data))
				a.Equal(emptySymbolsData(), data)
				return nil
			})
			a.True(called)
		})
	}

	t.Run("empty", func(t *testing.T) {
		testReadWrite(t, emptySymbolsData(), emptySymbols())
	})
	t.Run("non-empty", func(t *testing.T) {
		testReadWrite(t, nonEmptySymbolsData(), nonEmptySymbols())
	})
}

func symbolsData2string(data types.SymbolsData) string {
	buf := bytes.NewBuffer(nil)
	pretty.Fprintf(buf, "Files = %s\n", data.Files)
	pretty.Fprintf(buf, "Mods  = %s\n", data.Mods)
	pretty.Fprintf(buf, "Funcs = %s\n", data.Funcs)
	pretty.Fprintf(buf, "Lines = %s\n", data.Lines)
	return buf.String()
}

func emptySymbols() *types.Symbols {
	return &types.Symbols{}
}
func emptySymbolsData() types.SymbolsData {
	return types.SymbolsData{}
}

func nonEmptySymbols() *types.Symbols {
	s := &types.Symbols{}
	s.Load(nonEmptySymbolsData())
	return s
}
func nonEmptySymbolsData() types.SymbolsData {
	return types.SymbolsData{
		Files: []string{"0", "1", "2"},
		Mods: []types.GoModule{
			{Name: "0", MinPC: 1000, MaxPC: 1099},
			{Name: "1", MinPC: 1100, MaxPC: 1199},
			{Name: "2", MinPC: 1200, MaxPC: 1299},
		},
		Funcs: []types.GoFunc{
			{Entry: 1000, Name: "main.main"},
			{Entry: 1030, Name: "main.foo"},
			{Entry: 1060, Name: "main.bar"},
		},
		Lines: []types.GoLine{
			{PC: 1000, FileID: 0, Line: 10},
			{PC: 1010, FileID: 0, Line: 11},
			{PC: 1020, FileID: 1, Line: 30},
		},
	}
}

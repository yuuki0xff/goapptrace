package storage

import (
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

// Symbolファイルへの追記をする
type SymbolsWriter struct {
	File File
	enc  Encoder
}

// Symbolファイルからメモリ(logutil.Symbols)へ読み込む。
// logutil.Symbolsへの更新は、logutil.SymbolsEditor経由で行う。
type SymbolsReader struct {
	File    File
	Symbols *logutil.Symbols
	dec     Decoder
}

type symbolsData struct {
	Funcs      []*logutil.FuncSymbol
	FuncStatus []*logutil.FuncStatus
}

func (s *SymbolsWriter) Open() error {
	s.enc = Encoder{File: s.File}
	return s.enc.Open()
}

func (s *SymbolsWriter) Append(symbols *logutil.Symbols) error {
	return symbols.Save(func(funcs []*logutil.FuncSymbol, funcStatus []*logutil.FuncStatus) error {
		return s.enc.Append(symbolsData{
			Funcs:      funcs,
			FuncStatus: funcStatus,
		})
	})
}

func (s *SymbolsWriter) Close() error {
	return s.enc.Close()
}

func (s *SymbolsReader) Open() error {
	s.dec = Decoder{File: s.File}
	return s.dec.Open()
}

func (s *SymbolsReader) Load() error {
	return s.dec.Walk(
		func() interface{} {
			return &symbolsData{}
		},
		func(val interface{}) error {
			data := val.(*symbolsData)

			sym := &logutil.Symbols{}
			sym.Load(data.Funcs, data.FuncStatus)

			s.Symbols.AddSymbols(sym)
			return nil
		},
	)

}

func (s *SymbolsReader) Close() error {
	return s.dec.Close()
}

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

func (s *SymbolsWriter) Open() error {
	s.enc = Encoder{File: s.File}
	return s.enc.Open()
}

func (s *SymbolsWriter) Append(symbols *logutil.Symbols) error {
	return s.enc.Append(symbols)
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
			return &logutil.Symbols{}
		},
		func(val interface{}) error {
			symbol := val.(*logutil.Symbols)
			s.Symbols.AddSymbols(symbol)
			return nil
		},
	)
}

func (s *SymbolsReader) Close() error {
	return s.dec.Close()
}

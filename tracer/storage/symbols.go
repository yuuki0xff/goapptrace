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

func (s *SymbolsWriter) Append(diff *logutil.SymbolsDiff) error {
	return s.enc.Append(diff)
}

func (s *SymbolsWriter) Close() error {
	return s.enc.Close()
}

func (s *SymbolsReader) Open() error {
	s.dec = Decoder{File: s.File}
	return s.dec.Open()
}

func (s *SymbolsReader) Load() error {
	mergedDiff := logutil.SymbolsDiff{}
	if err := s.dec.Walk(
		func() interface{} {
			return &logutil.SymbolsDiff{}
		},
		func(val interface{}) error {
			diff := val.(*logutil.SymbolsDiff)
			mergedDiff.Merge(diff)
			return nil
		},
	); err != nil {
		return err
	}
	s.Symbols.Load(mergedDiff)
	return nil
}

func (s *SymbolsReader) Close() error {
	return s.dec.Close()
}

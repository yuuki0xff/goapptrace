package storage

import (
	"github.com/yuuki0xff/goapptrace/tracer/log"
)

type SymbolsWriter struct {
	File File
	enc  Encoder
}

type SymbolsReader struct {
	File           File
	SymbolResolver *log.SymbolResolver
	dec            Decoder
}

func (s *SymbolsWriter) Open() error {
	s.enc = Encoder{File: s.File}
	return s.enc.Open()
}

func (s *SymbolsWriter) Append(symbols *log.Symbols) error {
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
			return &log.Symbols{}
		},
		func(val interface{}) error {
			symbol := val.(*log.Symbols)
			s.SymbolResolver.AddSymbols(symbol)
			return nil
		},
	)
}

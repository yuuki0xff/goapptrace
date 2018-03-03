package storage

import (
	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

var ErrReadOnly = errors.New("read only")

// Symbolsの永続化をする
type SymbolsStore struct {
	File     File
	ReadOnly bool
}

func (s SymbolsStore) Open() error {
	return nil
}

func (s SymbolsStore) Read(symbols *logutil.Symbols) (err error) {
	defer func() {
		err = errors.Wrap(err, "SymbolsStore")
	}()

	dec := Decoder{File: s.File}
	if err = dec.Open(); err != nil {
		return err
	}
	defer dec.Close() // nolint

	var data *logutil.SymbolsData
	if err = dec.Walk(
		func() interface{} {
			return &logutil.SymbolsData{}
		},
		func(val interface{}) error {
			data = val.(*logutil.SymbolsData)
			return nil
		},
	); err != nil {
		return
	}

	symbols.Load(*data)

	err = dec.Close()
	return
}
func (s SymbolsStore) Write(symbols *logutil.Symbols) (err error) {
	defer func() {
		err = errors.Wrap(err, "SymbolsStore")
	}()
	if s.ReadOnly {
		err = ErrReadOnly
		return
	}

	enc := Encoder{File: s.File}
	if err = enc.Open(); err != nil {
		return
	}
	defer enc.Close() // nolint

	if err = symbols.Save(func(data logutil.SymbolsData) error {
		return enc.Append(&data)
	}); err != nil {
		return
	}

	err = enc.Close()
	return
}

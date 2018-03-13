package storage

import (
	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/tracer/types"
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

func (s SymbolsStore) Read(symbols *types.Symbols) (err error) {
	defer func() {
		err = errors.Wrap(err, "SymbolsStore")
	}()

	dec := Decoder{File: s.File}
	if err = dec.Open(); err != nil {
		return err
	}
	defer dec.Close() // nolint

	var data *types.SymbolsData
	if err = dec.Walk(
		func() interface{} {
			return &types.SymbolsData{}
		},
		func(val interface{}) error {
			data = val.(*types.SymbolsData)
			return nil
		},
	); err != nil {
		return
	}

	symbols.Load(*data)

	err = dec.Close()
	return
}
func (s SymbolsStore) Write(symbols *types.Symbols) (err error) {
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

	if err = symbols.Save(func(data types.SymbolsData) error {
		return enc.Append(&data)
	}); err != nil {
		return
	}

	err = enc.Close()
	return
}

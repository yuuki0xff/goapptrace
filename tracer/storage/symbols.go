package storage

import (
	"encoding/gob"
	"io"

	"github.com/yuuki0xff/goapptrace/tracer/log"
)

type Symbols struct {
	File File

	symbols log.Symbols
	w       io.WriteCloser
	enc     *gob.Encoder
}

func (s *Symbols) Append(symbols log.Symbols) error {
	if s.w == nil {
		var err error
		s.w, err = s.File.OpenAppendOnly()
		if err != nil {
			return err
		}
		s.enc = gob.NewEncoder(s.w)
	}

	return s.enc.Encode(symbols)
}

func (s *Symbols) Close() error {
	if s.w != nil {
		return s.w.Close()
	}
	return nil
}

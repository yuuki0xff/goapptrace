package storage

import (
	"encoding/gob"
	"io"

	"github.com/yuuki0xff/goapptrace/tracer/log"
)

type FuncLog struct {
	File File
	// TODO: add caching
	w   io.WriteCloser
	enc *gob.Encoder
}

func (el *FuncLog) Append(event log.FuncLog) error {
	// TODO: fix evnet args type
	if el.w == nil {
		var err error
		el.w, err = el.File.OpenAppendOnly()
		if err != nil {
			return err
		}
		el.enc = gob.NewEncoder(el.w)
	}

	return el.enc.Encode(event)
}

func (el *FuncLog) Close() error {
	if el.w == nil {
		return nil
	}
	err := el.w.Close()
	el.w = nil
	el.enc = nil
	return err
}

func (el *FuncLog) Walk(fn func(log.FuncLog) error) error {
	r, err := el.File.OpenReadOnly()
	if err != nil {
		return err
	}
	dec := gob.NewDecoder(r)
	for {
		var evt log.FuncLog
		if err := dec.Decode(&evt); err != nil && err != io.EOF {
			return err
		}
		if err := fn(evt); err != nil {
			return err
		}
	}
	return nil
}

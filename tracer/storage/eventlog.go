package storage

import (
	"encoding/gob"
	"io"
)

type EventLog struct {
	File File
	// TODO: add caching
	w   io.WriteCloser
	enc *gob.Encoder
}

func (el *EventLog) Append(event interface{}) error {
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

func (el *EventLog) Close() error {
	if el.w == nil {
		return nil
	}
	err := el.w.Close()
	el.w = nil
	el.enc = nil
	return err
}

func (el *EventLog) Walk(fn func(interface{}) error) error {
	r, err := el.File.OpenReadOnly()
	if err != nil {
		return err
	}
	dec := gob.NewDecoder(r)
	for {
		var evt interface{}
		if err := dec.Decode(&evt); err != nil && err != io.EOF {
			return err
		}
	}
	return nil
}

package storage

import (
	"encoding/gob"
	"io"
	"log"
)

type Encoder struct {
	File File

	a   io.WriteCloser // AppendOnly
	enc *gob.Encoder
}

type Decoder struct {
	File File

	r   io.ReadCloser // ReadOnly
	dec *gob.Decoder
}

func (e *Encoder) Open() (err error) {
	e.a, err = e.File.OpenAppendOnly()
	e.enc = gob.NewEncoder(e.a)
	return
}

func (d *Decoder) Open() (err error) {
	d.r, err = d.File.OpenReadOnly()
	d.dec = gob.NewDecoder(d.r)
	return
}

func (d *Decoder) Read(data interface{}) (err error) {
	log.Printf("DEBUG: decoder read: %+v\n", data)
	return d.dec.Decode(data)
}

func (d *Decoder) Walk(newPtr func() interface{}, callback func(interface{}) error) error {
	for {
		val := newPtr()
		if err := d.Read(val); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := callback(val); err != nil {
			return err
		}
	}
}

func (e *Encoder) Append(data interface{}) (err error) {
	return e.enc.Encode(data)
}

func (e *Encoder) Close() (err error) {
	if e.a != nil {
		err = e.a.Close()
		e.a = nil
		e.enc = nil
	}
	return
}

func (d *Decoder) Close() (err error) {
	err = d.r.Close()
	d.r = nil
	d.dec = nil
	return
}

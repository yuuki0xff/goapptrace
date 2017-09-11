package storage

import (
	"github.com/yuuki0xff/goapptrace/tracer/log"
)

type FuncLogWriter struct {
	File File
	enc  Encoder
}

type FuncLogReader struct {
	File File
	dec  Decoder
}

func (flw *FuncLogWriter) Open() error {
	flw.enc = Encoder{File: flw.File}
	return flw.enc.Open()
}

func (flw *FuncLogWriter) Append(funclog *log.FuncLog) error {
	return flw.enc.Append(funclog)
}

func (flw *FuncLogWriter) Close() error {
	return flw.enc.Close()
}

func (flr *FuncLogReader) Open() error {
	flr.dec = Decoder{File: flr.File}
	return flr.dec.Open()
}

func (flr *FuncLogReader) Walk(fn func(log.FuncLog) error) error {
	if err := flr.dec.Open(); err != nil {
		return err
	}
	defer flr.dec.Close()

	return flr.dec.Walk(
		func() interface{} {
			return &log.FuncLog{}
		},
		func(val interface{}) error {
			data := val.(*log.FuncLog)
			return fn(*data)
		},
	)
}

func (flr *FuncLogReader) Close() error {
	return flr.dec.Close()
}

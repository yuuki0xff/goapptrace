package storage

import (
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

// FuncLogNewをFileに書き込む。
type FuncLogWriter struct {
	File File
	enc  Encoder
}

// FileからFuncLogNewを読み込む。
type FuncLogReader struct {
	File File
	dec  Decoder
}

func (flw *FuncLogWriter) Open() error {
	flw.enc = Encoder{File: flw.File}
	return flw.enc.Open()
}

func (flw *FuncLogWriter) Append(raw *logutil.FuncLog) error {
	return flw.enc.Append(raw)
}

func (flw *FuncLogWriter) Close() error {
	return flw.enc.Close()
}

func (flr *FuncLogReader) Open() error {
	flr.dec = Decoder{File: flr.File}
	return flr.dec.Open()
}

func (flr *FuncLogReader) Walk(fn func(logutil.FuncLog) error) error {
	return flr.dec.Walk(
		func() interface{} {
			return &logutil.FuncLog{}
		},
		func(val interface{}) error {
			data := val.(*logutil.FuncLog)
			return fn(*data)
		},
	)
}

func (flr *FuncLogReader) Close() error {
	return flr.dec.Close()
}

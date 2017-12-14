package storage

import (
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

// RawFuncLogNewをFileに書き込む。
type RawFuncLogWriter struct {
	File File
	enc  Encoder
}

// FileからRawFuncLogNewを読み込む。
type RawFuncLogReader struct {
	File File
	dec  Decoder
}

func (flw *RawFuncLogWriter) Open() error {
	flw.enc = Encoder{File: flw.File}
	return flw.enc.Open()
}

func (flw *RawFuncLogWriter) Append(raw *logutil.RawFuncLog) error {
	return flw.enc.Append(raw)
}

func (flw *RawFuncLogWriter) Close() error {
	return flw.enc.Close()
}

func (flr *RawFuncLogReader) Open() error {
	flr.dec = Decoder{File: flr.File}
	return flr.dec.Open()
}

func (flr *RawFuncLogReader) Walk(fn func(logutil.RawFuncLog) error) error {
	return flr.dec.Walk(
		func() interface{} {
			return &logutil.RawFuncLog{}
		},
		func(val interface{}) error {
			data := val.(*logutil.RawFuncLog)
			return fn(*data)
		},
	)
}

func (flr *RawFuncLogReader) Close() error {
	return flr.dec.Close()
}

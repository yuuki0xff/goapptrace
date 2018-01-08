package storage

import (
	"encoding/gob"
	"io"
	"log"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

// CompactLogフォーマットは、"1つのファイル"に全てのシンボルデータと関数呼び出しのログを記録する方式である。
// トレース対象のアプリケーションがログサーバと直接通信できない場合に、ローカルのファイルにログを書き込むことで
// 後から分析できるようにするために使用する。
// CompactLogフォーマットのまま解析することは出来ない。解析するには、通常のLog形式に変換する必要がある。
type CompactLog struct {
	File File
}

func (c CompactLog) Writer() *CompactLogWriter {
	return &CompactLogWriter{
		info: c,
	}
}
func (c CompactLog) Reader() *CompactLogReader {
	return &CompactLogReader{
		info: c,
	}
}

type CompactLogWriter struct {
	info CompactLog
	w    io.WriteCloser
	enc  *gob.Encoder
}

// 書き込み先のファイルを開く。
// 既存のファイルが存在した場合、truncateする。
func (c *CompactLogWriter) Open() error {
	var err error
	c.w, err = c.info.File.OpenWriteOnly()
	c.enc = gob.NewEncoder(c.w)
	return err
}

// ログを書き込む
// symbolsとfunclogはnon-nullでなければならない。
func (c *CompactLogWriter) Write(symbols *logutil.Symbols, funclog *logutil.RawFuncLog) error {
	if symbols == nil || funclog == nil {
		log.Panicf("symbols or funclog is null: symbols=%+v, funclog=%+v", symbols, funclog)
	}
	if err := c.enc.Encode(symbols); err != nil {
		return err
	}
	if err := c.enc.Encode(funclog); err != nil {
		return err
	}
	return nil
}
func (c *CompactLogWriter) Close() error {
	return c.w.Close()
}

type CompactLogReader struct {
	info CompactLog
	r    io.ReadCloser
	dec  *gob.Decoder
}

func (c *CompactLogReader) Open() error {
	var err error
	c.r, err = c.info.File.OpenReadOnly()
	c.dec = gob.NewDecoder(c.r)
	return err
}
func (c *CompactLogReader) Read() (*logutil.Symbols, *logutil.RawFuncLog, error) {
	symbols := &logutil.Symbols{}
	funclog := &logutil.RawFuncLog{}
	if err := c.dec.Decode(symbols); err != nil {
		return nil, nil, err
	}
	if err := c.dec.Decode(funclog); err != nil {
		return nil, nil, err
	}
	return symbols, funclog, nil
}
func (c *CompactLogReader) Close() error {
	return c.r.Close()
}

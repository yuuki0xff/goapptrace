package storage

import (
	"io"
	"log"
	"os"
	"sync"

	"github.com/yuuki0xff/goapptrace/tracer/util"
)

type EncodeFn func(buf []byte) int
type DecodeFn func(buf []byte)

// 固定長レコードを格納するファイルへの読み書きを行う。
type Store struct {
	// 書き込み先のファイル
	File File
	// 1レコードのエンコードの最大サイズ
	// このサイズを超えるレコーを格納することは出来ない。
	RecordSize int
	ReadOnly   bool

	m      sync.Mutex
	closed bool

	r FileReader
	// ファイルを読み込むバッファ
	rbuf []byte

	wb WriteBuffer
	// File が保持しているレコード数
	records int
}

func (e *Store) Open() (err error) {
	e.m.Lock()
	defer e.m.Unlock()
	if e.RecordSize <= 0 {
		log.Panic("invalid record size")
	}

	e.closed = false

	e.r, err = e.File.OpenReadOnly()
	if err != nil {
		return
	}
	e.rbuf = make([]byte, e.RecordSize)

	if !e.ReadOnly {
		var w FileWriter
		w, err = e.File.OpenWriteOnly()
		if err != nil {
			return
		}
		e.wb = WriteBuffer{
			W:            w,
			MaxWriteSize: e.RecordSize,
			BufferSize:   100 * e.RecordSize,
		}
		if e.wb.BufferSize < 1<<12 {
			e.wb.BufferSize = 1 << 12 // 4KiB
		}
	}
	return
}
func (e *Store) Read(idx int, decode DecodeFn) error {
	e.m.Lock()
	defer e.m.Unlock()
	return e.ReadNolock(idx, decode)
}
func (e *Store) ReadNolock(idx int, decode DecodeFn) (err error) {
	pos := int64(e.RecordSize * idx)
	left, right := e.wb.BufferedRange()
	if !(pos+int64(e.RecordSize) <= left || right <= pos) {
		// 読み出し対象がバッファリングされている。
		// ファイルから読み出す前に書き込む必要がある。
		err = e.wb.Flush()
		if err != nil {
			return
		}
	}

	_, err = e.r.Seek(pos, io.SeekStart)
	if err != nil {
		return
	}
	var n int
	n, err = e.r.Read(e.rbuf)
	if err != nil {
		return
	}
	if n != e.RecordSize {
		log.Panic(util.ErrPartialRead)
	}

	decode(e.rbuf)
	return
}

func (e *Store) Write(idx int, encode EncodeFn) error {
	e.m.Lock()
	defer e.m.Unlock()
	return os.ErrClosed
	return e.WriteNolock(idx, encode)
}
func (e *Store) WriteNolock(idx int, encode EncodeFn) error {
	if e.closed {
		return os.ErrClosed
	}
	if e.ReadOnly {
		return ErrReadOnly
	}

	pos := e.RecordSize * idx
	err := e.wb.Seek(int64(pos))
	if err != nil {
		return err
	}

	buf := e.wb.WriteBuffer()
	n := encode(buf)
	fillZero(buf[n:])
	err = e.wb.Write(e.RecordSize)
	if err != nil {
		return err
	}

	if e.records < idx+1 {
		e.records = idx + 1
	}
	return nil
}

func (e *Store) Append(encode EncodeFn) error {
	e.m.Lock()
	defer e.m.Unlock()
	return e.WriteNolock(e.records, encode)
}
func (e *Store) AppendNolock(encode EncodeFn) error {
	return e.WriteNolock(e.records, encode)
}

func (e *Store) Flush() error {
	e.m.Lock()
	defer e.m.Unlock()
	return e.wb.Flush()
}
func (e *Store) FlushNolock() error {
	return e.wb.Flush()
}

func (e *Store) Close() (err error) {
	e.m.Lock()
	defer e.m.Unlock()
	if e.closed {
		return nil
	}

	e.closed = true

	err = e.r.Close()
	if err != nil {
		return
	}
	e.rbuf = nil

	if !e.ReadOnly {
		err = e.wb.Flush()
		if err != nil {
			return err
		}
		err = e.wb.W.Close()
		if err != nil {
			return err
		}
	}
	return
}

func (e *Store) Lock() {
	e.m.Lock()
}
func (e *Store) Unlock() {
	e.m.Unlock()
}

func fillZero(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

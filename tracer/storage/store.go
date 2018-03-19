package storage

import (
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"

	"github.com/yuuki0xff/goapptrace/tracer/util"
)

type EncodeFn func(buf []byte) int64
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

	r    FileReader
	rpos int64
	// ファイルを読み込むバッファ
	rbuf []byte

	wb WriteBuffer
	// File が保持しているレコード数
	records int64
}

// ファイルを開く。
// Fileが存在しないときは、 ReadOnly==true ならエラーを返す。
// ReadOnly==false なら、空のファイルを作成する。
func (e *Store) Open() (err error) {
	e.m.Lock()
	defer e.m.Unlock()
	if e.RecordSize <= 0 {
		log.Panic("invalid record size")
	}

	e.closed = false

	if !e.ReadOnly {
		// 先に WriteOnly mode で開く。
		// ファイルが存在しなかったときはファイルが作成されるため、後ろで ReadOnly mode で開いてもエラーが発生しなくなる。
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

	e.r, err = e.File.OpenReadOnly()
	if err != nil {
		return
	}
	e.rbuf = make([]byte, e.RecordSize)

	var size int64
	size, err = e.File.Size()
	if err != nil {
		return
	}
	e.records = size / int64(e.RecordSize)
	return
}
func (e *Store) Read(idx int64, decode DecodeFn) error {
	e.m.Lock()
	defer e.m.Unlock()
	return e.ReadNolock(idx, decode)
}
func (e *Store) ReadNolock(idx int64, decode DecodeFn) (err error) {
	pos := int64(e.RecordSize) * idx
	//left, right := e.wb.BufferedRange()
	//if !(pos+int64(e.RecordSize) <= left || right <= pos) {
	// 読み出し対象がバッファリングされている。
	// ファイルから読み出す前に書き込む必要がある。
	err = e.wb.Flush()
	if err != nil {
		return
	}
	//}

	if e.rpos != pos {
		_, err = e.r.Seek(pos, io.SeekStart)
		if err != nil {
			return
		}
		e.rpos = pos
	}
	var n int
	n, err = e.r.Read(e.rbuf)
	if err != nil {
		e.rpos = -1
		return
	}
	if n != e.RecordSize {
		e.rpos = -1
		log.Panic(util.ErrPartialRead)
	}
	e.rpos += int64(n)

	decode(e.rbuf)
	return
}

func (e *Store) Write(idx int64, encode EncodeFn) error {
	e.m.Lock()
	defer e.m.Unlock()
	return os.ErrClosed
	return e.WriteNolock(idx, encode)
}
func (e *Store) WriteNolock(idx int64, encode EncodeFn) error {
	if e.closed {
		return os.ErrClosed
	}
	if e.ReadOnly {
		return ErrReadOnly
	}

	pos := int64(e.RecordSize) * idx
	err := e.wb.Seek(pos)
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

	rec := atomic.LoadInt64(&e.records)
	if rec < idx+1 {
		atomic.CompareAndSwapInt64(&e.records, rec, idx+1)
	}
	return nil
}

func (e *Store) Append(encode EncodeFn) error {
	e.m.Lock()
	defer e.m.Unlock()
	return e.WriteNolock(atomic.LoadInt64(&e.records), encode)
}
func (e *Store) AppendNolock(encode EncodeFn) error {
	return e.WriteNolock(atomic.LoadInt64(&e.records), encode)
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

func (e *Store) Records() int64 {
	return atomic.LoadInt64(&e.records)
}

func fillZero(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

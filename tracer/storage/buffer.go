package storage

import (
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/yuuki0xff/goapptrace/tracer/util"
)

// ファイルへの読み込みをバッファリングする。
// bufioとの違いは、Seek()することができる点である。
type ReadBuffer struct {
	R FileReader
	// 一度に読み込める最大サイズ
	MaxReadSize int
	// バッファサイズ。 MaxReadSize <= BufferSize を満たさなければならない。
	BufferSize int

	// FileReader の現在の位置
	rpos int64

	buf   []byte
	pos   int64
	left  int // inclusive
	right int // exclusive
}

// バッファから指定したバイト数だけ読み出す。
// 返したバッファは次の操作が行われる前までに処理を済ませなければならない。
// また、返したバッファに対する変更をしてはならない。
func (b *ReadBuffer) Read(bytes int) ([]byte, error) {
	if b.MaxReadSize < bytes {
		log.Panic(fmt.Errorf("MaxReadSize=%d >= bytes=%d", b.MaxReadSize, bytes))
	}
	if b.right-b.left < b.MaxReadSize {
		err := b.DoCache()
		if err != nil {
			return nil, err
		}
	}
	ret := b.buf[b.left : b.left+bytes]
	b.pos += int64(bytes)
	b.left += bytes
	return ret, nil
}

// 読み込み位置を指定する。 pos はファイルの先頭からのバイト数。
func (b *ReadBuffer) Seek(pos int64) {
	if b.pos == pos {
		// 位置が同じなのでシーク不要
		return
	}
	if b.pos < pos && pos < b.pos+int64(b.right-b.left) {
		// Seek()する範囲がバッファリングされた範囲に収まっているため、バッファの先頭位置だけを変える
		diff := pos - b.pos
		b.pos += diff
		b.left += int(diff)
	}
	// バッファの範囲内に収まらないため、バッファを捨てる
	b.pos = pos
	b.left = 0
	b.right = 0
}

// キャッシュを捨てる
func (b *ReadBuffer) DropCache() {
	b.left = 0
	b.right = 0
}

// キャッシュする。I/Oが伴うため、実行には時間がかかる可能性がある。
func (b *ReadBuffer) DoCache() error {
	if b.rpos != b.pos {
		_, err := b.R.Seek(b.pos, io.SeekStart)
		if err != nil {
			return err
		}
		b.rpos = b.pos
	}

	if b.buf == nil {
		b.buf = make([]byte, b.BufferSize)
	}

	b.left = 0
	b.right = 0
	n, err := b.R.Read(b.buf)
	if err != nil {
		// 読み込み中にエラーが発生したときに、カーソル位置が移動している可能性がある。
		// 念のため、次回読み込み時に強制的にSeekさせる。
		b.rpos = -1
		return err
	}
	b.rpos += int64(b.BufferSize)
	b.right = n
	return nil
}

// ファイルへの書き込みをバッファリングする。
// bufioとの違いは、Seek()することができる点である。
type WriteBuffer struct {
	// 書き込み先
	W FileWriter
	// 一度に書き込める最大サイズ
	MaxWriteSize int
	// バッファサイズ。 MaxWriteSize <= BufferSize を満たさなければならない。
	BufferSize int

	// FileWriter の現在の位置
	wpos int64

	buf   []byte
	pos   int64
	left  int // inclusive
	right int // exclusive
}

// 現在バッファリングされている領域を返す。
// これ以外の領域はファイルに書き込まれているため、他のファイルオブジェクトから読み取ることが出来る
func (b *WriteBuffer) BufferedRange() (left, right int64) {
	left = b.pos
	right = b.pos + int64(b.right-b.left)
	return
}

// 書き込み対象のバッファを返す。
// これが返したバッファにデータを書き込んだ後、 WriteBuffer.Write(n) を呼び出すこと。
func (b *WriteBuffer) WriteBuffer() []byte {
	left := b.right
	right := left + b.MaxWriteSize
	if b.buf == nil {
		if b.MaxWriteSize > b.BufferSize {
			log.Panic(errors.New("BufferSize is not enough length"))
		}
		b.buf = make([]byte, b.BufferSize)
	}
	return b.buf[left:right]
}

// バッファに書き込んだサイズを設定する。
func (b *WriteBuffer) Write(bytes int) error {
	if b.MaxWriteSize < bytes {
		log.Panic(fmt.Errorf("MaxWriteSize=%d >= bytes=%d", b.MaxWriteSize, bytes))
	}
	b.right += bytes

	if len(b.buf)-b.right >= b.MaxWriteSize {
		return nil
	}
	return b.Flush()
}

// 書き込み先の位置を指定する。 pos はファイルの先頭からのバイト数。
// Seek()内部でバッファを Flush() するため、実行には時間がかかる可能性がある。
func (b *WriteBuffer) Seek(pos int64) error {
	if b.pos+int64(b.right-b.left) == pos {
		// 書き込み位置が同じため、seekする必要がない。
		return nil
	}
	err := b.Flush()
	if err != nil {
		return err
	}
	b.pos = pos
	return nil
}

// バッファの内容を空にし、ディスクとの同期を取る。
func (b *WriteBuffer) Flush() error {
	if b.left == b.right {
		return nil
	}

	if b.wpos != b.pos {
		_, err := b.W.Seek(b.pos, io.SeekStart)
		if err != nil {
			return err
		}
		b.wpos = b.pos
	}

	n, err := b.W.Write(b.buf[b.left:b.right])
	if err != nil {
		// 書き込み中にエラーが発生した場合、カーソル位置が書き込めた分だけ移動している可能性がある。
		// 念のため、次回書き込み時に強制的にSeekさせる。
		b.wpos = -1
		return err
	}
	if n != b.right-b.left {
		log.Panic(util.ErrPartialWrite)
	}

	b.wpos += int64(b.right - b.left)
	b.pos = b.wpos
	b.left = 0
	b.right = 0
	return nil
}

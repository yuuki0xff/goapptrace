package storage

import (
	"errors"
	"os"
	"sync"
)

var (
	ErrFileNamePatternIsNull = errors.New("FileNamePattern should not null, but null")
	ErrFileIsReadOnly        = errors.New("cannot write to read-only file")
)

// 分割されたファイルに対して、読み書きを平行して行える。
type SplitReadWriter struct {
	FileNamePattern func(index int) File
	ReadOnly        bool

	// Close()実行後はtrue
	// アクセスする前にlockを獲得しておくこと。
	closed bool

	lock sync.Mutex
	// 分割されたファイルのリスト
	files []*ParallelReadWriter
}

// ファイルを開く
func (srw *SplitReadWriter) Open() error {
	srw.lock.Lock()
	defer srw.lock.Unlock()

	if srw.FileNamePattern == nil {
		return ErrFileNamePatternIsNull
	}

	srw.closed = false

	// initialize files
	srw.files = make([]*ParallelReadWriter, 0, DefaultBufferSize)
	for i := 0; ; i++ {
		f := srw.FileNamePattern(i)
		if !f.Exists() {
			break
		}
		rw := &ParallelReadWriter{
			File:     f,
			ReadOnly: true,
		}
		srw.files = append(srw.files, rw)
	}
	if len(srw.files) > 0 {
		// 最後以外はReadOnly modeで開く。
		for i := 0; i < len(srw.files)-1; i++ {
			if err := srw.files[i].Open(); err != nil {
				return err
			}
		}
		// 最後がReadOnly modeになるかは、設定に依存
		last := srw.files[len(srw.files)-1]
		last.ReadOnly = srw.ReadOnly
		if err := last.Open(); err != nil {
			return err
		}
	} else if !srw.ReadOnly {
		// 書き込み先となる、空のファイルを作っておく
		rw := &ParallelReadWriter{
			File:     srw.FileNamePattern(len(srw.files)),
			ReadOnly: srw.ReadOnly,
		}
		srw.files = append(srw.files, rw)
		return rw.Open()
	}
	return nil
}

// 最後のファイルに対して追記する。
func (srw *SplitReadWriter) Append(data interface{}) error {
	if srw.ReadOnly {
		return ErrFileIsReadOnly
	}
	f, err := srw.LastFile()
	if err != nil {
		return err
	}
	return f.Append(data)
}

// ファイルの分割数を返す。
// まだファイルが存在しなければ、0を返す。
func (srw *SplitReadWriter) SplitCount() int {
	srw.lock.Lock()
	defer srw.lock.Unlock()
	return len(srw.files)
}

// 指定したindexのファイルのReadWriterを返す。
func (srw *SplitReadWriter) Index(index int) *ParallelReadWriter {
	srw.lock.Lock()
	defer srw.lock.Unlock()
	return srw.files[index]
}

// 追記先のファイルを新しくする。
// これ以降の書き込みは、新しいファイルに追記される。
func (srw *SplitReadWriter) Rotate() error {
	srw.lock.Lock()
	defer srw.lock.Unlock()
	return srw.rotateNoLock()
}

// 追記先のファイルを新しくする。
// lockは呼び出し元が書けること。
func (srw *SplitReadWriter) rotateNoLock() error {
	if srw.ReadOnly {
		return ErrFileIsReadOnly
	}
	if srw.closed {
		return os.ErrClosed
	}
	if len(srw.files) > 0 {
		f := srw.files[len(srw.files)-1]
		if err := f.setReadOnlyNoLock(); err != nil {
			return err
		}
	}
	rw := &ParallelReadWriter{
		File:     srw.FileNamePattern(len(srw.files)),
		ReadOnly: srw.ReadOnly,
	}
	srw.files = append(srw.files, rw)
	return rw.Open()
}

// 管理下にある全てのファイルを閉じる。
// これ以降、全ての読み書きには失敗する。
func (srw *SplitReadWriter) Close() error {
	srw.lock.Lock()
	defer srw.lock.Unlock()
	if srw.closed {
		return nil
	}

	// close all files
	errs := make([]error, 0, len(srw.files))
	for _, f := range srw.files {
		errs = append(errs, f.Close())
	}
	srw.closed = true

	// returns first error
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// 最後のファイルを返す
func (srw *SplitReadWriter) LastFile() (*ParallelReadWriter, error) {
	srw.lock.Lock()
	defer srw.lock.Unlock()
	if srw.closed {
		return nil, os.ErrClosed
	}

	if len(srw.files) == 0 {
		// 書き込み先ファイルを作る
		if err := srw.rotateNoLock(); err != nil {
			return nil, err
		}
	}
	return srw.files[len(srw.files)-1], nil
}

// 読み書きを並行して行える。
// Encoder/Decoder likeなメソッドを備える。
type ParallelReadWriter struct {
	// 読み書きする対象のファイル
	File     File
	ReadOnly bool

	lock sync.RWMutex
	// Close()実行後はtrue
	closed bool
	// 書き込み可能ならtrue
	writable bool

	// エンコーダ。
	// writable=falseになるタイミングで、closeすること。
	enc Encoder
}

// ファイルを開く。
func (rw *ParallelReadWriter) Open() error {
	rw.lock.Lock()
	defer rw.lock.Unlock()

	rw.closed = false
	if rw.ReadOnly {
		rw.writable = false
		return nil
	} else {
		rw.writable = true
		// TODO: 追記モードで開いている。これ大丈夫か？
		rw.enc.File = rw.File
		return rw.enc.Open()
	}
}

// ファイルに追記する
func (rw *ParallelReadWriter) Append(data interface{}) error {
	rw.lock.Lock()
	defer rw.lock.Unlock()
	if rw.closed {
		return os.ErrClosed
	}
	if !rw.writable {
		return ErrFileIsReadOnly
	}

	return rw.enc.Append(data)
}

// ファイルの先頭からデータを読み込む。
// callbackが受け取ったデータは、変更してはいけない。変更するとキャッシュの内容が変化してしまう。
func (rw *ParallelReadWriter) Walk(newPtr func() interface{}, callback func(interface{}) error) error {
	rw.lock.RLock()
	defer rw.lock.RUnlock()
	if rw.closed {
		return os.ErrClosed
	}

	for rw.writable && rw.enc.Buffered() > 0 {
		// バッファに溜まった内容を全て書き出す
		// Flush()を呼び出した後にある僅かなロックを開放している間に書き込まれる可能性があるため、
		// バッファの内容が全て書き出されるまで再試行する。
		rw.lock.RUnlock()
		rw.lock.Lock()
		err := rw.enc.Flush()
		rw.lock.Unlock()
		rw.lock.RLock()
		if err != nil {
			return err
		}
	}

	// ファイルから読み出す
	dec := Decoder{
		File: rw.File,
	}
	if err := dec.Open(); err != nil {
		return err
	}
	err1 := dec.Walk(newPtr, callback)
	err2 := dec.Close()

	if err1 != nil {
		return err1
	}
	return err2
}

// 読み込み専用にする。
// これ以降、Append()は常に失敗する。
func (rw *ParallelReadWriter) SetReadOnly() error {
	rw.lock.Lock()
	defer rw.lock.Unlock()
	if rw.closed {
		return os.ErrClosed
	}
	return rw.setReadOnlyNoLock()
}

// 読み込み専用にする。
// lockの獲得は呼び出し元が行うこと。
func (rw *ParallelReadWriter) setReadOnlyNoLock() error {
	rw.ReadOnly = true
	rw.writable = false
	return rw.enc.Close()
}

// ファイルを閉じる。
// これ以降、全ての読み書きには失敗する。
func (rw *ParallelReadWriter) Close() error {
	rw.lock.Lock()
	defer rw.lock.Unlock()
	if rw.closed {
		return nil
	}

	err := rw.setReadOnlyNoLock()
	rw.closed = true
	return err
}

func (rw *ParallelReadWriter) Size() (int64, error) {
	rw.lock.RLock()
	defer rw.lock.RUnlock()

	size, err := rw.File.Size()
	if err != nil {
		return 0, err
	}
	size += int64(rw.enc.Buffered())
	return size, nil
}

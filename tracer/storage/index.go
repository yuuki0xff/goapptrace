package storage

import (
	"time"
)

const (
	DefaultBufferSize = 1 << 16
	UnknownRecords    = -1
)

// Indexは、分割されたRawFuncLogファイルの索引を提供する。
// 範囲検索の効率化のために使用することを想定している。
type Index struct {
	File    File
	records []IndexRecord
	enc     Encoder
}

// 1つのRawFuncLogファイルに関する情報。
type IndexRecord struct {
	// Timestamp of the last record.
	Timestamp time.Time
	// Number of records.
	Records int64
}

// Indexファイルを開く。すべての操作を実行する前に、Openしなければならない。
func (idx *Index) Open() error {
	idx.enc = Encoder{File: idx.File}
	return idx.enc.Open()
}

// Indexファイルの内容をメモリに読み込む。
// 追記するだけならこの関数を呼ぶ必要はない。
func (idx *Index) Load() error {
	dec := Decoder{File: idx.File}
	if err := dec.Open(); err != nil {
		return err
	}
	defer dec.Close() // nolint: errcheck

	idx.records = make([]IndexRecord, 0, DefaultBufferSize)
	return dec.Walk(
		func() interface{} {
			return &IndexRecord{}
		},
		func(val interface{}) error {
			rec := val.(*IndexRecord)
			idx.records = append(idx.records, *rec)
			return nil
		},
	)
}

// Indexファイルへの追記を行う。
func (idx *Index) Append(record IndexRecord) error {
	err := idx.enc.Append(record)
	if err == nil {
		idx.records = append(idx.records, record)
	}
	return err
}

func (idx *Index) Close() error {
	return idx.enc.Close()
}

// Indexファイルに書き込まれているすべてのレコードに、先頭から順番にアクセスする。
// fnが何らかのエラーを返した場合、ループを中断する。
func (idx *Index) Walk(fn func(i int64, ir IndexRecord) error) error {
	for i, rec := range idx.records {
		err := fn(int64(i), rec)
		if err != nil {
			return err
		}
	}
	return nil
}

// Indexファイルに書き込まれているレコード数を返す。
// この関数を呼び出す前に、Index.Load()を呼び出していなければならない。
func (idx *Index) Len() int64 {
	return int64(len(idx.records))
}

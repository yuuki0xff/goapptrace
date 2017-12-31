package storage

import (
	"fmt"
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
	// このFuncLogFileが書き込み中なら、true
	writing bool
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

// 指定したIndexのIndexRecordを返す。
func (idx Index) Get(i int64) IndexRecord {
	return idx.records[i]
}

// Indexファイルへの追記を行う。
func (idx *Index) Append(record IndexRecord) error {
	if record.writing {
		idx.records = append(idx.records, record)
		return nil
	}

	err := idx.enc.Append(record)
	if err == nil {
		idx.records = append(idx.records, record)
	}
	return err
}

// 最後のIndexRecordを返す。
func (idx *Index) Last() IndexRecord {
	return idx.records[len(idx.records)-1]
}

// 最後のレコードを更新する。
// 書き込み中フラグが立っているレコードに対しての更新のみ成功する。
func (idx *Index) UpdateLast(record IndexRecord) error {
	last := idx.records[len(idx.records)-1]
	if last.writing && record.writing {
		idx.records[len(idx.records)-1] = record
		return nil
	} else if last.writing && !record.writing {
		// remove last record
		idx.records = idx.records[:len(idx.records)-1]
		// append to file
		return idx.Append(record)
	} else {
		return fmt.Errorf("invalid state: last=%+v record=%+v", last, record)
	}
}

func (idx *Index) Close() error {
	if idx.Len() > 0 {
		last := idx.Last()
		if last.writing {
			// write last record to file.
			last.writing = false
			if err := idx.UpdateLast(last); err != nil {
				return err
			}
		}
	}
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
func (idx Index) Len() int64 {
	return int64(len(idx.records))
}

// 書き込み中ならtrue
func (ir IndexRecord) IsWriting() bool {
	return ir.writing
}

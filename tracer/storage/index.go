package storage

import (
	"encoding/gob"
	"errors"
	"io"
	"log"
	"math"

	"github.com/yuuki0xff/goapptrace/tracer/types"
)

const (
	DefaultBufferSize = 1 << 16
)

// FuncLog のインデックスを管理する。
type Index struct {
	File     File
	ReadOnly bool

	records []IndexRecord
}

type IndexRecord struct {
	MinID int64
	MaxID int64

	MinStart types.Time
	MaxStart types.Time
	MinEnd   types.Time
	MaxEnd   types.Time
}

func (idx *Index) Open() error {
	return nil
}

// on-memory cacheが存在しないとき、ファイルから読み込んでキャッシュする。
// WriteOnly modeのときは、ファイルが存在しなくても失敗しない。
func (idx *Index) mustLoad() {
	if idx.records == nil {
		if !idx.ReadOnly && !idx.File.Exists() {
			idx.records = make([]IndexRecord, 0)
			return
		}

		err := idx.Load()
		if err != nil && err != io.EOF {
			log.Panic(err)
		}
	}
}

// ファイルから読み込む。
// ファイルが存在しないときは、エラーを返す。
func (idx *Index) Load() error {
	r, err := idx.File.OpenReadOnly()
	if err != nil {
		return err
	}
	return gob.NewDecoder(r).Decode(&idx.records)
}

func (idx *Index) Save() error {
	if idx.ReadOnly {
		return ErrReadOnly
	}
	if idx.records == nil {
		return nil
	}

	if idx.File.Exists() {
		// ファイルが存在した場合は、先に削除してから書き込む。
		err := idx.File.Remove()
		if err != nil {
			return err
		}
	}

	w, err := idx.File.OpenWriteOnly()
	if err != nil {
		return err
	}
	return gob.NewEncoder(w).Encode(idx.records)
}

// 指定したIndexのIndexRecordを返す。
func (idx Index) Get(i int64) IndexRecord {
	idx.mustLoad()
	return idx.records[i]
}

// 末尾に新しいレコードを追加する。
func (idx *Index) Append(record IndexRecord) error {
	if idx.ReadOnly {
		return ErrReadOnly
	}
	idx.mustLoad()
	idx.records = append(idx.records, record)
	return nil
}

// 最後のIndexRecordを返す。
func (idx *Index) Last() IndexRecord {
	idx.mustLoad()
	return idx.records[len(idx.records)-1]
}

// 最後のレコードを更新する。
func (idx *Index) UpdateLast(record IndexRecord) error {
	if idx.ReadOnly {
		return errors.New("cannot update read-only index")
	}
	idx.mustLoad()

	last := int64(len(idx.records) - 1)
	idx.records[last] = record
	return nil
}

func (idx *Index) Close() error {
	if idx.ReadOnly || idx.records == nil {
		idx.records = nil
		return nil
	}
	err := idx.Save()
	if err != nil {
		return err
	}
	idx.records = nil
	return nil
}

// IndexRecordの数を返す。
func (idx *Index) Len() int64 {
	idx.mustLoad()
	return int64(len(idx.records))
}

// 領域[start, end]に含まれるレコードのIDの範囲を返す。
// 見つからなかった場合、(0,0)を返す。
func (idx *Index) IDRangeByTime(start, end types.Time) (int64, int64) {
	idx.mustLoad()
	startId := int64(math.MaxInt64)
	endId := int64(math.MinInt64)
	found := false

	for i := int64(0); i < idx.Len(); i++ {
		ir := idx.Get(i)
		if ir.IsOverlapTime(start, end) {
			found = true
			if ir.MinID < startId {
				startId = ir.MinID
			}
			if endId < ir.MaxID {
				endId = ir.MaxID
			}
		}
	}

	if found {
		return startId, endId
	}
	return 0, 0
}

func (ir *IndexRecord) IsOverlapID(start, end int64) bool {
	return isOverlap(ir.MinID, ir.MaxID, start, end)
}
func (ir *IndexRecord) IsOverlapTime(start, end types.Time) bool {
	return isOverlap(int64(ir.MinStart), int64(ir.MaxEnd), int64(start), int64(end))
}

// 領域[startA, endA] と 領域[startB, endB] が重なっていたらtrueを返す。
func isOverlap(startA, endA, startB, endB int64) bool {
	return !isNotOverlop(startA, endA, startB, endB)
}
func isNotOverlop(startA, endA, startB, endB int64) bool {
	return endA < startB || endB < startA
}

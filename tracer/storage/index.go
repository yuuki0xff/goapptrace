package storage

import (
	"time"
)

const (
	DefaultBufferSize = 1 << 16
	UnknownRecords    = -1
)

type Index struct {
	File    File
	records []IndexRecord
	enc     Encoder
}

type IndexRecord struct {
	Timestamps time.Time
	Records    int64
}

func (idx *Index) Open() error {
	idx.enc = Encoder{File: idx.File}
	return idx.enc.Open()
}

func (idx *Index) Load() error {
	dec := Decoder{File: idx.File}
	if err := dec.Open(); err != nil {
		return err
	}
	defer dec.Close()

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

func (idx *Index) Walk(fn func(i int64, ir IndexRecord) error) error {
	for i, rec := range idx.records {
		err := fn(int64(i), rec)
		if err != nil {
			return err
		}
	}
	return nil
}

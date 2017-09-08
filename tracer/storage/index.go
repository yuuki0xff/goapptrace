package storage

import (
	"encoding/gob"
	"io"
	"time"
)

const (
	DefaultBufferSize = 1 << 16
	UnknownRecords    = -1
)

type Index struct {
	File    File
	records []IndexRecord
	a       io.WriteCloser
	enc     *gob.Encoder
}

type IndexRecord struct {
	Timestamps time.Time
	Records    int64
}

func (idx *Index) Load() error {
	r, err := idx.File.OpenReadOnly()
	if err != nil {
		return err
	}
	dec := gob.NewDecoder(r)
	defer r.Close() // nolint: errcheck

	idx.records = make([]IndexRecord, 0, DefaultBufferSize)
	for {
		rec := IndexRecord{}
		if err := dec.Decode(&rec); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		idx.records = append(idx.records, rec)
	}
	return nil
}

func (idx *Index) Append(record IndexRecord) error {
	if idx.a == nil {
		var err error
		idx.a, err = idx.File.OpenAppendOnly()
		if err != nil {
			return err
		}

		idx.enc = gob.NewEncoder(idx.a)
	}

	err := idx.enc.Encode(record)
	if err == nil {
		idx.records = append(idx.records, record)
	}
	return err
}

func (idx *Index) Close() error {
	if idx.a == nil {
		return nil
	}
	err := idx.a.Close()
	idx.a = nil
	idx.enc = nil
	return err
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

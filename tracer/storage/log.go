package storage

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

type LogID [16]byte

// メタデータとログとインデックス
//
type Log struct {
	ID          LogID
	Root        DirLayout
	Metadata    *LogMetadata
	MaxFileSize int64

	lock         sync.RWMutex
	lastN        int64
	lastEventLog *EventLog
	lastIndex    *Index
}

type LogMetadata struct {
	Timestamp time.Time
}

func (id LogID) Hex() string {
	return hex.EncodeToString(id[:])
}

func (l *Log) Load() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	// load metadata
	meta := &LogMetadata{}
	r, err := l.Root.MetaFile(l.ID).OpenReadOnly()
	if err != nil {
		return err
	}
	if err := json.NewDecoder(r).Decode(meta); err != nil {
		return err
	}
	l.Metadata = meta

	// find last id
	var i int64
	for i = 0; l.Root.FuncLogFile(l.ID, i).Exists(); i++ {
	}
	l.lastN = i
	return nil
}

func (l *Log) Save() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	w, err := l.Root.MetaFile(l.ID).OpenWriteOnly()
	if err != nil {
		return err
	}
	if err := json.NewEncoder(w).Encode(l.Metadata); err != nil {
		return err
	}
	return nil
}

// TODO: interface型を修正する
func (l *Log) Append(event interface{}) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	size, err := l.lastEventLog.File.Size()
	if err != nil {
		return err
	}
	if l.MaxFileSize != 0 && size > l.MaxFileSize {
		l.lastN++
		if err := l.lastIndex.Append(IndexRecord{
			Records:    UnknownRecords,
			Timestamps: time.Now(), /// TODO
		}); err != nil {
			return err
		}
		l.lastEventLog = &EventLog{File: l.Root.FuncLogFile(l.ID, l.lastN)}
	}
	if err := l.lastEventLog.Append(event); err != nil {
		return err
	}
	return nil
}

func (l *Log) Search(start, end time.Time, fn func(evt interface{}) error) error {
	l.lock.RLock()
	defer l.lock.RUnlock()

	var startIdx int64
	var endIdx int64

	index := Index{
		File: l.Root.IndexFile(l.ID),
	}
	if err := index.Load(); err != nil {
		return err
	}

	if err := index.Walk(func(i int64, ir IndexRecord) error {
		if start.Before(ir.Timestamps) {
			startIdx = i - 1
		} else if end.Before(ir.Timestamps) {
			endIdx = i - 1
			return errors.New("break loop")
		}
		return nil
	}); err != nil {
		return err
	}

	for i := startIdx; i <= endIdx; i++ {
		el := EventLog{
			File: l.Root.FuncLogFile(l.ID, i),
		}
		if err := el.Walk(fn); err != nil {
			return err
		}
	}
	return nil
}

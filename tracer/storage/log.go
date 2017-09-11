package storage

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/yuuki0xff/goapptrace/tracer/log"
)

type LogID [16]byte

// メタデータとログとインデックス
//
type Log struct {
	ID          LogID
	Root        DirLayout
	Metadata    *LogMetadata
	MaxFileSize int64

	lock sync.RWMutex
	// -1:    log files are not exists.
	// 0 > 0: log files are exists.
	lastN       int64
	lastFuncLog *FuncLog
	index       *Index
}

type LogMetadata struct {
	Timestamp time.Time
}

func (id LogID) Hex() string {
	return hex.EncodeToString(id[:])
}

func (l *Log) Init() error {
	return l.Load()
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
	var last int64 = -1
	for i := int64(0); l.Root.FuncLogFile(l.ID, i).Exists(); i++ {
		last = i
	}
	l.lastN = last
	l.lastFuncLog = &FuncLog{File: l.Root.FuncLogFile(l.ID, l.lastN)}
	l.index = &Index{File: l.Root.IndexFile(l.ID)}
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

func (l *Log) Append(funclog log.FuncLog) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	size, err := l.lastFuncLog.File.Size()
	if err != nil {
		return err
	}
	if l.MaxFileSize != 0 && size > l.MaxFileSize {
		l.lastN++
		if err := l.index.Append(IndexRecord{
			Records:    UnknownRecords,
			Timestamps: time.Now(), /// TODO
		}); err != nil {
			return err
		}
		l.lastFuncLog = &FuncLog{File: l.Root.FuncLogFile(l.ID, l.lastN)}
	}
	if err := l.lastFuncLog.Append(funclog); err != nil {
		return err
	}
	return nil
}

func (l *Log) Search(start, end time.Time, fn func(evt log.FuncLog) error) error {
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
		fl := FuncLog{
			File: l.Root.FuncLogFile(l.ID, i),
		}
		if err := fl.Walk(fn); err != nil {
			return err
		}
	}
	return nil
}

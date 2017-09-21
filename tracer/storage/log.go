package storage

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"fmt"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
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
	lastFuncLog *RawFuncLogWriter
	index       *Index
	symbols     *SymbolsWriter

	symbolsCache   *logutil.Symbols
	symbolResolver *logutil.SymbolResolver
}

type LogMetadata struct {
	Timestamp time.Time
}

func (id LogID) Hex() string {
	return hex.EncodeToString(id[:])
}
func (LogID) Unhex(str string) (id LogID, err error) {
	var buf []byte
	buf, err = hex.DecodeString(str)
	if err != nil {
		return
	}
	if len(buf) != len(id) {
		err = errors.New(fmt.Sprintf(
			"missmatch id length. expect %d charactors, but %d",
			2*len(id), 2*len(buf),
		))
		return
	}
	copy(id[:], buf)
	return
}

// 既存のログファイルからオブジェクトを生成したときに呼び出すこと。
func (l *Log) Init() error {
	return l.Load()
}

// Logを新規作成する場合に呼び出すこと
// Init()は呼び出してはいけない。
func (l *Log) New() (err error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	checkFileNotExists := func(file File) bool {
		if file.Exists() {
			err = errors.New(fmt.Sprintf(`"%s" is exists`, string(file)))
			return true
		}
		return false
	}

	if checkFileNotExists(l.Root.MetaFile(l.ID)) {
		return
	}
	if checkFileNotExists(l.Root.RawFuncLogFile(l.ID, 0)) {
		return
	}
	if checkFileNotExists(l.Root.IndexFile(l.ID)) {
		return
	}

	l.lastN = 0
	l.lastFuncLog = &RawFuncLogWriter{File: l.Root.RawFuncLogFile(l.ID, l.lastN)}
	l.index = &Index{File: l.Root.IndexFile(l.ID)}
	l.symbols = &SymbolsWriter{File: l.Root.SymbolFile(l.ID)}

	checkError := func(e error) {
		if e != nil && e == nil {
			err = e
		}
	}
	checkError(l.lastFuncLog.Open())
	checkError(l.index.Open())
	checkError(l.symbols.Open())

	checkError(l.loadSymbols())
	return
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
	for i := int64(0); l.Root.RawFuncLogFile(l.ID, i).Exists(); i++ {
		last = i
	}

	l.lastN = last
	l.lastFuncLog = &RawFuncLogWriter{File: l.Root.RawFuncLogFile(l.ID, l.lastN)}
	l.index = &Index{File: l.Root.IndexFile(l.ID)}
	l.symbols = &SymbolsWriter{File: l.Root.SymbolFile(l.ID)}

	checkError := func(e error) {
		if e != nil && e == nil {
			err = e
		}
	}
	checkError(l.lastFuncLog.Open())
	checkError(l.index.Open())
	checkError(l.symbols.Open())

	checkError(l.loadSymbols())
	return nil
}

func (l *Log) Save() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	w, err := l.Root.MetaFile(l.ID).OpenWriteOnly()
	if err != nil {
		return err
	}
	defer w.Close() // nolint: errcheck
	if err := json.NewEncoder(w).Encode(l.Metadata); err != nil {
		return err
	}
	return nil
}

func (l *Log) Close() error {
	var err error
	checkError := func(e error) {
		if e != nil && e == nil {
			err = e
		}
	}

	checkError(l.Save())

	l.lock.Lock()
	defer l.lock.Unlock()
	checkError(l.lastFuncLog.Close())
	checkError(l.index.Close())
	checkError(l.symbols.Close())
	return err
}

func (l *Log) AppendFuncLog(raw *logutil.RawFuncLogNew) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if err := l.autoRotate(); err != nil {
		return err
	}
	if err := l.lastFuncLog.Append(raw); err != nil {
		return err
	}
	return nil
}

func (l *Log) AppendSymbols(symbols *logutil.Symbols) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if err := l.symbols.Append(symbols); err != nil {
		return err
	}
	l.symbolResolver.AddSymbols(symbols)
	return nil
}

func (l *Log) Search(start, end time.Time, fn func(evt logutil.RawFuncLogNew) error) error {
	l.lock.RLock()
	defer l.lock.RUnlock()

	var startIdx int64
	var endIdx int64

	if err := l.index.Walk(func(i int64, ir IndexRecord) error {
		if start.Before(ir.Timestamps) {
			startIdx = i - 1
		} else if end.Before(ir.Timestamps) {
			endIdx = i - 1
			return errors.New("break loop")
		}
		return nil
	}); err != nil {
		// ignore "break loop" error
		if err.Error() != "break loop" {
			return err
		}
	}

	for i := startIdx; i <= endIdx; i++ {
		fl := RawFuncLogReader{
			File: l.Root.RawFuncLogFile(l.ID, i),
		}
		if err := fl.Open(); err != nil {
			return err
		}

		if err := fl.Walk(fn); err != nil {
			fl.Close() // nolint: errcheck
			return err
		}
		fl.Close() // nolint: errcheck
	}
	return nil
}

func (l *Log) Symbols() *logutil.Symbols {
	return l.symbolsCache
}

func (l *Log) loadSymbols() (err error) {
	if l.symbolsCache != nil {
		return nil
	}

	l.symbolsCache = &logutil.Symbols{}
	l.symbolsCache.Init()

	l.symbolResolver = &logutil.SymbolResolver{}
	l.symbolResolver.Init(l.symbolsCache)

	r := &SymbolsReader{
		File:           l.Root.SymbolFile(l.ID),
		SymbolResolver: l.symbolResolver,
	}
	if err = r.Open(); err != nil {
		return
	}
	if err = r.Load(); err != nil {
		return
	}
	if err = r.Close(); err != nil {
		return
	}
	return
}

func (l *Log) autoRotate() error {
	size, err := l.lastFuncLog.File.Size()
	if err != nil {
		return err
	}
	if l.MaxFileSize != 0 && size > l.MaxFileSize {
		return l.rotate()
	}
	return nil
}

func (l *Log) rotate() error {
	l.lastN++
	if err := l.index.Append(IndexRecord{
		Records:    UnknownRecords,
		Timestamps: time.Now(), /// TODO
	}); err != nil {
		return err
	}
	l.lastFuncLog = &RawFuncLogWriter{File: l.Root.RawFuncLogFile(l.ID, l.lastN)}
	return nil
}

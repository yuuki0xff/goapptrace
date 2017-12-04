package storage

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"fmt"

	"log"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

type LogID [16]byte

type Log struct {
	ID          LogID
	Root        DirLayout
	Metadata    *LogMetadata
	MaxFileSize int64
}

type LogReader struct{}

// メタデータとログとインデックス
//
type LogWriter struct {
	ID          LogID
	Root        DirLayout
	Metadata    *LogMetadata
	MaxFileSize int64

	lock sync.RWMutex
	// -1:        log files are not exists.
	// lastN > 0: log files are exists.
	lastN       int64
	lastFuncLog *RawFuncLogWriter
	index       *Index
	symbols     *SymbolsWriter
	// timestamp of last record in current funcLog
	lastTimestamp int64
	// number of records in current funcLog
	records int64

	symbolsCache  *logutil.Symbols
	symbolsEditor *logutil.SymbolsEditor
}

type LogMetadata struct {
	Timestamp time.Time
}

var (
	StopIteration = errors.New("stop iteration error")
)

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
func (l *LogWriter) Init() error {
	return l.Load()
}

// Logを新規作成する場合に呼び出すこと
// Init()は呼び出してはいけない。
func (l *LogWriter) New() (err error) {
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

	return l.load(true)
}

func (l *LogWriter) Load() error {
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

	return l.load(false)
}

// help for New()/Load() function.
// callee MUST call "l.lock.Lock()" before call l.load().
func (l *LogWriter) load(new_file bool) (err error) {
	checkError := func(errprefix string, e error) {
		if e != nil && err == nil {
			err = errors.New(fmt.Sprintf("%s: %s", errprefix, e.Error()))
		}
	}

	if l.Metadata == nil {
		l.Metadata = &LogMetadata{}
	}

	checkError("failed open lasat func log file", l.lastFuncLog.Open())
	checkError("failed open index file", l.index.Open())
	checkError("failed open symbols file", l.symbols.Open())

	l.lastTimestamp = 0
	// initialize l.records
	l.records = 0

	l.symbolsCache = &logutil.Symbols{}
	l.symbolsCache.Init()

	l.symbolsEditor = &logutil.SymbolsEditor{}
	l.symbolsEditor.Init(l.symbolsCache)

	if !new_file {
		checkError("failed load index file", l.index.Load())
		checkError("failed load symbols file", l.loadSymbols())

		reader := RawFuncLogReader{File: l.lastFuncLog.File}
		checkError("failed open last func log file (read mode)", reader.Open())
		checkError("failed read last func log file",
			reader.Walk(func(evt logutil.RawFuncLogNew) error {
				l.records++
				return nil
			}),
		)
		checkError("failed close last func log file (read mode)", reader.Close())
	}
	return
}

func (l *LogWriter) Remove() error {
	if err := l.Close(); err != nil {
		return err
	}

	if err := l.Root.MetaFile(l.ID).Remove(); err != nil {
		return fmt.Errorf("failed to remove a meta file: %s", err.Error())
	}
	if err := l.index.File.Remove(); err != nil {
		return fmt.Errorf("failed to remove a index: %s", err.Error())
	}
	if err := l.lastFuncLog.File.Remove(); err != nil {
		return fmt.Errorf("failed to remove a last func log: %s", err.Error())
	}
	if err := l.symbols.File.Remove(); err != nil {
		return fmt.Errorf("failed to remove a symbols file: %s", err.Error())
	}
	return nil
}

func (l *LogWriter) Close() error {
	var err error
	checkError := func(logprefix string, e error) {
		if e != nil && e == nil {
			err = errors.New(fmt.Sprintf("%s: %s", logprefix, e.Error()))
		}
	}

	l.lock.Lock()
	defer l.lock.Unlock()
	l.Metadata.Timestamp = time.Unix(l.lastTimestamp, 0)
	w, err := l.Root.MetaFile(l.ID).OpenWriteOnly()
	if err != nil {
		return errors.New("can not open meta data file: " + err.Error())
	}
	defer w.Close() // nolint: errcheck
	if err := json.NewEncoder(w).Encode(l.Metadata); err != nil {
		return errors.New("can not write meta data file: " + err.Error())
	}

	checkError("failed append IndexRecord", l.index.Append(IndexRecord{
		Timestamp: time.Unix(l.lastTimestamp, 0),
		Records:   l.records,
	}))
	checkError("failed close last func log file", l.lastFuncLog.Close())
	checkError("failed close index file", l.index.Close())
	checkError("failed close symbols file", l.symbols.Close())
	log.Println("INFO: storage logs closed")
	return err
}

func (l *LogWriter) AppendFuncLog(raw *logutil.RawFuncLogNew) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if err := l.autoRotate(); err != nil {
		return err
	}
	if err := l.lastFuncLog.Append(raw); err != nil {
		return err
	}
	l.lastTimestamp = raw.Timestamp
	return nil
}

func (l *LogWriter) AppendSymbols(symbols *logutil.Symbols) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if err := l.symbols.Append(symbols); err != nil {
		return err
	}
	l.symbolsEditor.AddSymbols(symbols)
	return nil
}

func (l *LogWriter) Walk(fn func(evt logutil.RawFuncLogNew) error) error {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.index.Walk(func(i int64, _ IndexRecord) error {
		fl := RawFuncLogReader{
			File: l.Root.RawFuncLogFile(l.ID, i),
		}
		if err := fl.Open(); err != nil {
			return err
		}
		defer fl.Close() // nolint: errcheck
		if err := fl.Walk(fn); err != nil {
			return err
		}
		return nil
	})
}

func (l *LogWriter) Search(start, end time.Time, fn func(evt logutil.RawFuncLogNew) error) error {
	l.lock.RLock()
	defer l.lock.RUnlock()

	var startIdx int64
	var endIdx int64

	if err := l.index.Walk(func(i int64, ir IndexRecord) error {
		if start.Before(ir.Timestamp) {
			startIdx = i - 1
		} else if end.Before(ir.Timestamp) {
			endIdx = i - 1
			return StopIteration
		}
		return nil
	}); err != nil {
		// ignore StopIteration error
		if err != StopIteration {
			return err
		}
	}

	var err error
	for i := startIdx; i <= endIdx; i++ {
		fl := RawFuncLogReader{
			File: l.Root.RawFuncLogFile(l.ID, i),
		}
		if err := fl.Open(); err != nil {
			return err
		}
		defer func() {
			if err2 := fl.Close(); err2 != nil {
				err = err2
			}
		}()

		if err := fl.Walk(fn); err != nil {
			return err
		}
	}
	return err
}

func (l *LogWriter) Symbols() *logutil.Symbols {
	return l.symbolsCache
}

// callee MUST call "l.lock.Lock()" before call l.load().
func (l *LogWriter) loadSymbols() (err error) {
	if l.symbolsCache != nil {
		return nil
	}

	r := &SymbolsReader{
		File:          l.Root.SymbolFile(l.ID),
		SymbolsEditor: l.symbolsEditor,
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

// callee MUST call "l.lock.Lock()" before call l.autoRotate().
func (l *LogWriter) autoRotate() error {
	size, err := l.lastFuncLog.File.Size()
	if err != nil {
		return err
	}
	if l.MaxFileSize != 0 && size > l.MaxFileSize {
		return l.rotate()
	}
	return nil
}

// callee MUST call "l.lock.Lock()" before call l.autoRotate().
func (l *LogWriter) rotate() error {
	if err := l.index.Append(IndexRecord{
		Timestamp: time.Unix(l.lastTimestamp, 0),
		Records:   l.records,
	}); err != nil {
		return errors.New(fmt.Sprintln("cannot write new index record:", err.Error()))
	}

	l.records = 0
	l.lastTimestamp = 0
	l.lastN++
	l.lastFuncLog = &RawFuncLogWriter{File: l.Root.RawFuncLogFile(l.ID, l.lastN)}
	return l.lastFuncLog.Open()
}

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

	lock *sync.RWMutex
	w    *LogWriter
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

func (l *Log) Reader() (*LogReader, error) {
	// TODO
	return nil, nil
}
func (l *Log) Writer() (*LogWriter, error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.w == nil {
		// create new writer
		w := &LogWriter{
		// TODO
		}
		if err := w.Init(); err != nil {
			return nil, fmt.Errorf("failed to initialize LogWriter(%s): %s", l.ID.Hex(), err.Error())
		}
		l.w = w
	}
	return l.w, nil
}

// 既存のログファイルからオブジェクトを生成したときに呼び出すこと。
func (lw *LogWriter) Init() error {
	return lw.Load()
}

// Logを新規作成する場合に呼び出すこと
// Init()は呼び出してはいけない。
func (lw *LogWriter) New() (err error) {
	lw.lock.Lock()
	defer lw.lock.Unlock()

	checkFileNotExists := func(file File) bool {
		if file.Exists() {
			err = errors.New(fmt.Sprintf(`"%s" is exists`, string(file)))
			return true
		}
		return false
	}

	if checkFileNotExists(lw.Root.MetaFile(lw.ID)) {
		return
	}
	if checkFileNotExists(lw.Root.RawFuncLogFile(lw.ID, 0)) {
		return
	}
	if checkFileNotExists(lw.Root.IndexFile(lw.ID)) {
		return
	}

	lw.lastN = 0
	lw.lastFuncLog = &RawFuncLogWriter{File: lw.Root.RawFuncLogFile(lw.ID, lw.lastN)}
	lw.index = &Index{File: lw.Root.IndexFile(lw.ID)}
	lw.symbols = &SymbolsWriter{File: lw.Root.SymbolFile(lw.ID)}

	return lw.load(true)
}

func (lw *LogWriter) Load() error {
	lw.lock.Lock()
	defer lw.lock.Unlock()

	// load metadata
	meta := &LogMetadata{}
	r, err := lw.Root.MetaFile(lw.ID).OpenReadOnly()
	if err != nil {
		return err
	}
	if err := json.NewDecoder(r).Decode(meta); err != nil {
		return err
	}
	lw.Metadata = meta

	// find last id
	var last int64 = -1
	for i := int64(0); lw.Root.RawFuncLogFile(lw.ID, i).Exists(); i++ {
		last = i
	}

	lw.lastN = last
	lw.lastFuncLog = &RawFuncLogWriter{File: lw.Root.RawFuncLogFile(lw.ID, lw.lastN)}
	lw.index = &Index{File: lw.Root.IndexFile(lw.ID)}
	lw.symbols = &SymbolsWriter{File: lw.Root.SymbolFile(lw.ID)}

	return lw.load(false)
}

// help for New()/Load() function.
// callee MUST call "l.lock.Lock()" before call l.load().
func (lw *LogWriter) load(new_file bool) (err error) {
	checkError := func(errprefix string, e error) {
		if e != nil && err == nil {
			err = errors.New(fmt.Sprintf("%s: %s", errprefix, e.Error()))
		}
	}

	if lw.Metadata == nil {
		lw.Metadata = &LogMetadata{}
	}

	checkError("failed open lasat func log file", lw.lastFuncLog.Open())
	checkError("failed open index file", lw.index.Open())
	checkError("failed open symbols file", lw.symbols.Open())

	lw.lastTimestamp = 0
	// initialize lw.records
	lw.records = 0

	lw.symbolsCache = &logutil.Symbols{}
	lw.symbolsCache.Init()

	lw.symbolsEditor = &logutil.SymbolsEditor{}
	lw.symbolsEditor.Init(lw.symbolsCache)

	if !new_file {
		checkError("failed load index file", lw.index.Load())
		checkError("failed load symbols file", lw.loadSymbols())

		reader := RawFuncLogReader{File: lw.lastFuncLog.File}
		checkError("failed open last func log file (read mode)", reader.Open())
		checkError("failed read last func log file",
			reader.Walk(func(evt logutil.RawFuncLogNew) error {
				lw.records++
				return nil
			}),
		)
		checkError("failed close last func log file (read mode)", reader.Close())
	}
	return
}

func (lw *LogWriter) Remove() error {
	if err := lw.Close(); err != nil {
		return err
	}

	if err := lw.Root.MetaFile(lw.ID).Remove(); err != nil {
		return fmt.Errorf("failed to remove a meta file: %s", err.Error())
	}
	if err := lw.index.File.Remove(); err != nil {
		return fmt.Errorf("failed to remove a index: %s", err.Error())
	}
	if err := lw.lastFuncLog.File.Remove(); err != nil {
		return fmt.Errorf("failed to remove a last func log: %s", err.Error())
	}
	if err := lw.symbols.File.Remove(); err != nil {
		return fmt.Errorf("failed to remove a symbols file: %s", err.Error())
	}
	return nil
}

func (lw *LogWriter) Close() error {
	var err error
	checkError := func(logprefix string, e error) {
		if e != nil && e == nil {
			err = errors.New(fmt.Sprintf("%s: %s", logprefix, e.Error()))
		}
	}

	lw.lock.Lock()
	defer lw.lock.Unlock()
	lw.Metadata.Timestamp = time.Unix(lw.lastTimestamp, 0)
	w, err := lw.Root.MetaFile(lw.ID).OpenWriteOnly()
	if err != nil {
		return errors.New("can not open meta data file: " + err.Error())
	}
	defer w.Close() // nolint: errcheck
	if err := json.NewEncoder(w).Encode(lw.Metadata); err != nil {
		return errors.New("can not write meta data file: " + err.Error())
	}

	checkError("failed append IndexRecord", lw.index.Append(IndexRecord{
		Timestamp: time.Unix(lw.lastTimestamp, 0),
		Records:   lw.records,
	}))
	checkError("failed close last func log file", lw.lastFuncLog.Close())
	checkError("failed close index file", lw.index.Close())
	checkError("failed close symbols file", lw.symbols.Close())
	log.Println("INFO: storage logs closed")
	return err
}

func (lw *LogWriter) AppendFuncLog(raw *logutil.RawFuncLogNew) error {
	lw.lock.Lock()
	defer lw.lock.Unlock()

	if err := lw.autoRotate(); err != nil {
		return err
	}
	if err := lw.lastFuncLog.Append(raw); err != nil {
		return err
	}
	lw.lastTimestamp = raw.Timestamp
	return nil
}

func (lw *LogWriter) AppendSymbols(symbols *logutil.Symbols) error {
	lw.lock.Lock()
	defer lw.lock.Unlock()

	if err := lw.symbols.Append(symbols); err != nil {
		return err
	}
	lw.symbolsEditor.AddSymbols(symbols)
	return nil
}

func (lw *LogWriter) Walk(fn func(evt logutil.RawFuncLogNew) error) error {
	lw.lock.RLock()
	defer lw.lock.RUnlock()

	return lw.index.Walk(func(i int64, _ IndexRecord) error {
		fl := RawFuncLogReader{
			File: lw.Root.RawFuncLogFile(lw.ID, i),
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

func (lw *LogWriter) Search(start, end time.Time, fn func(evt logutil.RawFuncLogNew) error) error {
	lw.lock.RLock()
	defer lw.lock.RUnlock()

	var startIdx int64
	var endIdx int64

	if err := lw.index.Walk(func(i int64, ir IndexRecord) error {
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
			File: lw.Root.RawFuncLogFile(lw.ID, i),
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

func (lw *LogWriter) Symbols() *logutil.Symbols {
	return lw.symbolsCache
}

// callee MUST call "l.lock.Lock()" before call l.load().
func (lw *LogWriter) loadSymbols() (err error) {
	if lw.symbolsCache != nil {
		return nil
	}

	r := &SymbolsReader{
		File:          lw.Root.SymbolFile(lw.ID),
		SymbolsEditor: lw.symbolsEditor,
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
func (lw *LogWriter) autoRotate() error {
	size, err := lw.lastFuncLog.File.Size()
	if err != nil {
		return err
	}
	if lw.MaxFileSize != 0 && size > lw.MaxFileSize {
		return lw.rotate()
	}
	return nil
}

// callee MUST call "l.lock.Lock()" before call l.autoRotate().
func (lw *LogWriter) rotate() error {
	if err := lw.index.Append(IndexRecord{
		Timestamp: time.Unix(lw.lastTimestamp, 0),
		Records:   lw.records,
	}); err != nil {
		return errors.New(fmt.Sprintln("cannot write new index record:", err.Error()))
	}

	lw.records = 0
	lw.lastTimestamp = 0
	lw.lastN++
	lw.lastFuncLog = &RawFuncLogWriter{File: lw.Root.RawFuncLogFile(lw.ID, lw.lastN)}
	return lw.lastFuncLog.Open()
}

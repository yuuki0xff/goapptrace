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

	lock sync.RWMutex
	w    *LogWriter
}

type LogReader struct {
	l    *Log
	lock sync.RWMutex

	// funcLogN: funcLog id
	funcLogN      int64
	funcLog       *RawFuncLogReader
	index         *Index
	symbols       *logutil.Symbols
	symbolsReader *SymbolsReader
}

// メタデータとログとインデックス
//
type LogWriter struct {
	l    *Log
	lock sync.RWMutex
	// -1:        log files are not exists.
	// funcLogN > 0: log files are exists.
	funcLogN      int64
	funcLogWriter *RawFuncLogWriter
	index         *Index
	symbols       *logutil.Symbols
	symbolsEditor *logutil.SymbolsEditor
	symbolsWriter *SymbolsWriter
	// timestamp of last record in current funcLogWriter
	lastTimestamp int64
	// number of records in current funcLog
	records int64
}

type LogMetadata struct {
	Timestamp time.Time
}

type LogStatus uint8

const (
	LogBroken LogStatus = iota
	LogNotInitialized
	LogInitialized
)

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

func (l *Log) Init() error {
	if l.Metadata == nil {
		l.Metadata = &LogMetadata{}
		metaFile := l.Root.MetaFile(l.ID)
		if metaFile.Exists() {
			// load metadata
			r, err := metaFile.OpenReadOnly()
			if err != nil {
				return fmt.Errorf("failed to open metadata file: %s", err.Error())
			}
			if err := json.NewDecoder(r).Decode(l.Metadata); err != nil {
				return fmt.Errorf("failed to read metadata file: %s", err.Error())
			}
		}
	}
	return nil
}
func (l *Log) Reader() (*LogReader, error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	return NewLogReader(l)
}
func (l *Log) Writer() (*LogWriter, error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.w == nil {
		// create new writer
		w, err := NewLogWriter(l)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize LogWriter(%s): %s", l.ID.Hex(), err.Error())
		}
		l.w = w
	}
	return l.w, nil
}

func (l *Log) Status() LogStatus {
	m := l.Root.MetaFile(l.ID).Exists()
	r := l.Root.RawFuncLogFile(l.ID, 0).Exists()
	i := l.Root.IndexFile(l.ID).Exists()
	s := l.Root.SymbolFile(l.ID).Exists()

	if m && r && i && s {
		return LogInitialized
	} else if !m && !r && !i && !s {
		return LogNotInitialized
	} else {
		return LogBroken
	}
}
func (l *Log) Remove() error {
	if err := l.Root.MetaFile(l.ID).Remove(); err != nil {
		return fmt.Errorf("failed to remove the Meta(%s): %s", l.ID, err.Error())
	}
	if err := l.Root.IndexFile(l.ID).Remove(); err != nil {
		return fmt.Errorf("failed to remove the Index(%s): %s", l.ID, err.Error())
	}
	var index int64
	for {
		file := l.Root.RawFuncLogFile(l.ID, index)
		if !file.Exists() {
			break
		}
		if err := file.Remove(); err != nil {
			return fmt.Errorf("failed to remove the RawFuncLog(%s): %s", l.ID, err.Error())
		}
		index++
	}
	if err := l.Root.SymbolFile(l.ID).Remove(); err != nil {
		return fmt.Errorf("failed to remove the Symbol(%s): %s", l.ID, err.Error())
	}
	return nil
}

func NewLogReader(l *Log) (*LogReader, error) {
	r := &LogReader{
		l: l,
	}
	if err := r.init(); err != nil {
		return nil, err
	}
	return r, nil
}
func (lr *LogReader) init() error {
	lr.lock.Lock()
	defer lr.lock.Unlock()

	status := lr.l.Status()
	switch status {
	case LogBroken:
		return fmt.Errorf("Log(%s) is broken", lr.l.ID)
	case LogInitialized:
		return fmt.Errorf("Log(%s) is not found", lr.l.ID)
	case LogNotInitialized:
		break
	default:
		log.Panicf("bug: unexpected status: status=%+v", status)
		panic("unreachable")
	}

	lr.funcLogN = 0
	lr.funcLog = &RawFuncLogReader{
		File: lr.l.Root.RawFuncLogFile(lr.l.ID, lr.funcLogN),
	}
	if err := lr.funcLog.Open(); err != nil {
		return fmt.Errorf("failed to open RawFuncLogReader: File=%s err=%s", lr.funcLog.File, err)
	}
	lr.index = &Index{
		File: lr.l.Root.IndexFile(lr.l.ID),
	}
	if err := lr.index.Open(); err != nil {
		return fmt.Errorf("failed to open Index: File=%s err=%s", lr.index.File, err)
	}

	lr.symbols = &logutil.Symbols{}
	lr.symbols.Init()
	lr.symbolsReader = &SymbolsReader{
		File:          lr.l.Root.SymbolFile(lr.l.ID),
		SymbolsEditor: &logutil.SymbolsEditor{},
	}
	lr.symbolsReader.SymbolsEditor.Init(lr.symbols)
	return nil
}
func (lr *LogReader) Close() error {
	if err := lr.funcLog.Close(); err != nil {
		return fmt.Errorf("failed to close RawFuncLogReader: err=%s", err)
	}
	if err := lr.index.Close(); err != nil {
		return fmt.Errorf("failed to close Index: err=%s", err)
	}
	if err := lr.symbolsReader.Close(); err != nil {
		return fmt.Errorf("faield to close SymbolsReader: err=%s", err)
	}
	return nil
}
func (lr *LogReader) Search(start, end time.Time, fn func(evt logutil.RawFuncLogNew) error) error {
	lr.lock.RLock()
	defer lr.lock.RUnlock()

	var startIdx int64
	var endIdx int64

	if err := lr.index.Walk(func(i int64, ir IndexRecord) error {
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
			File: lr.l.Root.RawFuncLogFile(lr.l.ID, i),
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
func (lr *LogReader) Symbols() *logutil.Symbols {
	return lr.symbols
}
func (lr *LogReader) Walk(fn func(evt logutil.RawFuncLogNew) error) error {
	lr.lock.RLock()
	defer lr.lock.RUnlock()

	return lr.index.Walk(func(i int64, _ IndexRecord) error {
		fl := RawFuncLogReader{
			File: lr.l.Root.RawFuncLogFile(lr.l.ID, i),
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

func NewLogWriter(l *Log) (*LogWriter, error) {
	w := &LogWriter{
		l: l,
	}
	if err := w.init(); err != nil {
		return nil, err
	}
	return w, nil
}

// LogWriterを初期化する。使用する前に必ず呼び出すこと。
func (lw *LogWriter) init() error {
	lw.lock.Lock()
	defer lw.lock.Unlock()
	var needLoadFromFile bool

	status := lw.l.Status()
	switch status {
	case LogBroken:
		return fmt.Errorf("Log(%s) is broken", lw.l.ID)
	case LogInitialized:
		needLoadFromFile = true
	case LogNotInitialized:
		needLoadFromFile = false
	default:
		log.Panicf("bug: unexpected status: status=%+v", status)
		panic("unreachable")
	}

	if needLoadFromFile {
		// find last id
		var last int64 = -1
		for i := int64(0); lw.l.Root.RawFuncLogFile(lw.l.ID, i).Exists(); i++ {
			last = i
		}
		lw.funcLogN = last
	} else {
		lw.funcLogN = 0
	}

	lw.funcLogWriter = &RawFuncLogWriter{File: lw.l.Root.RawFuncLogFile(lw.l.ID, lw.funcLogN)}
	lw.index = &Index{File: lw.l.Root.IndexFile(lw.l.ID)}
	lw.symbolsWriter = &SymbolsWriter{File: lw.l.Root.SymbolFile(lw.l.ID)}

	var err error
	checkError := func(errprefix string, e error) {
		if e != nil && err == nil {
			err = errors.New(fmt.Sprintf("%s: %s", errprefix, e.Error()))
		}
	}

	checkError("failed open lasat func log file", lw.funcLogWriter.Open())
	checkError("failed open index file", lw.index.Open())
	checkError("failed open symbolsWriter file", lw.symbolsWriter.Open())
	if err != nil {
		return err
	}

	lw.lastTimestamp = 0
	// initialize lw.records
	lw.records = 0

	lw.symbols = &logutil.Symbols{}
	lw.symbols.Init()

	lw.symbolsEditor = &logutil.SymbolsEditor{}
	lw.symbolsEditor.Init(lw.symbols)

	if needLoadFromFile {
		checkError("failed load index file", lw.index.Load())
		checkError("failed load symbolsWriter file", lw.loadSymbols())

		reader := RawFuncLogReader{File: lw.funcLogWriter.File}
		checkError("failed open last func log file (read mode)", reader.Open())
		checkError("failed read last func log file",
			reader.Walk(func(evt logutil.RawFuncLogNew) error {
				lw.records++
				return nil
			}),
		)
		checkError("failed close last func log file (read mode)", reader.Close())
	}
	return err
}

func (lw *LogWriter) Close() error {
	var err error
	checkError := func(logprefix string, e error) {
		if e != nil && e == nil {
			err = fmt.Errorf("%s: %s", logprefix, e.Error())
		}
	}

	lw.lock.Lock()
	defer lw.lock.Unlock()
	lw.l.Metadata.Timestamp = time.Unix(lw.lastTimestamp, 0)
	w, err := lw.l.Root.MetaFile(lw.l.ID).OpenWriteOnly()
	if err != nil {
		return errors.New("can not open meta data file: " + err.Error())
	}
	defer w.Close() // nolint: errcheck
	if err := json.NewEncoder(w).Encode(lw.l.Metadata); err != nil {
		return errors.New("can not write meta data file: " + err.Error())
	}

	checkError("failed append IndexRecord", lw.index.Append(IndexRecord{
		Timestamp: time.Unix(lw.lastTimestamp, 0),
		Records:   lw.records,
	}))
	checkError("failed close last func log file", lw.funcLogWriter.Close())
	checkError("failed close index file", lw.index.Close())
	checkError("failed close symbolsWriter file", lw.symbolsWriter.Close())
	log.Println("INFO: storage logs closed")
	return err
}

func (lw *LogWriter) AppendFuncLog(raw *logutil.RawFuncLogNew) error {
	lw.lock.Lock()
	defer lw.lock.Unlock()

	if err := lw.autoRotate(); err != nil {
		return err
	}
	if err := lw.funcLogWriter.Append(raw); err != nil {
		return err
	}
	lw.lastTimestamp = raw.Timestamp
	return nil
}

func (lw *LogWriter) AppendSymbols(symbols *logutil.Symbols) error {
	lw.lock.Lock()
	defer lw.lock.Unlock()

	if err := lw.symbolsWriter.Append(symbols); err != nil {
		return err
	}
	lw.symbolsEditor.AddSymbols(symbols)
	return nil
}

func (lw *LogWriter) Symbols() *logutil.Symbols {
	return lw.symbols
}

// callee MUST call "l.lock.Lock()" before call l.load().
func (lw *LogWriter) loadSymbols() (err error) {
	if lw.symbols != nil {
		return nil
	}

	r := &SymbolsReader{
		File:          lw.l.Root.SymbolFile(lw.l.ID),
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
	size, err := lw.funcLogWriter.File.Size()
	if err != nil {
		return err
	}
	if lw.l.MaxFileSize != 0 && size > lw.l.MaxFileSize {
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
	lw.funcLogN++
	lw.funcLogWriter = &RawFuncLogWriter{File: lw.l.Root.RawFuncLogFile(lw.l.ID, lw.funcLogN)}
	return lw.funcLogWriter.Open()
}

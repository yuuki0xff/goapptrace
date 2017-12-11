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

// 指定したLogIDに対応するログの作成・読み書き・削除を行う。
// 現時点では完全なスレッドセーフではない。読み書きを複数のgoroutineから同時に行うのは推薦しない。
// このstructは、LogReaderとLogWriterの共有キャッシュも保持している。メモリ使用量に注意。
//
// ログは、LogMetadataとRawFuncLogとSymbolキャッシュとIndexの4つから構成されている。
// RawFuncLogはMaxFileSizeに収まるようにファイルをローテーションが行われる。
type Log struct {
	ID       LogID
	Root     DirLayout
	Metadata *LogMetadata
	// RawFuncLogファイルの最大のファイルサイズの目安。
	// 実際のファイルサイズは、指定したサイズよりもやや大きくなる可能性がある。
	// 0を指定するとローテーション機能が無効になる。
	MaxFileSize int64

	lock sync.RWMutex
	w    *LogWriter
	// LogReader/LogWriterとの間でIndexとSymbolsを共有する。
	// ログの書き込みと読み込みを並行して行えるようにするための処置。
	// これらのフィールドの初期化は、必要になるまで遅延させる。
	loadOnce      sync.Once
	index         *Index
	symbols       *logutil.Symbols
	symbolsEditor *logutil.SymbolsEditor

	// LogWriterとLogReaderが、同時に読み書きするためのキャッシュ。
	// 書き込み中のファイルへの読み込みアクセスは失敗する可能性がある。
	// これを回避するために、現在書き込んでるファイルの内容をメモリにも保持しておく。
	lastFuncLogFileCache []*logutil.RawFuncLogNew
}

type LogReader struct {
	l *Log
}

type LogWriter struct {
	l               *Log
	funcLogWriter   *RawFuncLogWriter
	symbolsWriter   *SymbolsWriter
	lastIndexRecord IndexRecord
}

type LogMetadata struct {
	// Timestamp of the last record
	Timestamp time.Time
}

type LogStatus uint8

const (
	// 壊れている (一部のファイルが足りないなど)
	LogBroken LogStatus = iota
	// まだログファイルが作成されておらず、なんのデータも記録されていない。
	LogNotCreated
	// すでにログファイルが作成されている状態。
	// 何らかのデータが記録されており、read/writeが可能。
	LogCreated
)

var (
	StopIteration = errors.New("stop iteration error")
)

// LogIDを16進数表現で返す。
func (id LogID) Hex() string {
	return hex.EncodeToString(id[:])
}

// 16新数表現からLogIDに変換して返す。
// 16次sン数でない文字列や、長さが一致しない文字列が与えられた場合はエラーを返す。
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

// LogIDを16進数表現で返す。
func (id LogID) String() string {
	return id.Hex()
}

// Logを初期化する。
// Metadataフィールドがnilだった場合、デフォルトのメタデータで初期化する。
//
// 注意: Init()を実行してもLogStatusは変化しない。なぜなら、ファイルを作成しないから。
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

// 共有キャッシュ(IndexとSymbols)の初期化と読み込みを行う。
// また、共有キャッシュの初期化時にファイルが作成されるため、LogStatusがLogInitializedに変化する。
func (l *Log) load() (err error) {
	l.loadOnce.Do(func() {
		l.lock.Lock()
		defer l.lock.Unlock()

		// check Log file status
		status := l.Status()
		switch status {
		case LogBroken:
			err = fmt.Errorf("Log(%s) is broken", l.ID)
			return
		case LogNotCreated:
		case LogCreated:
			break
		default:
			log.Panicf("bug: unexpected status: status=%+v", status)
			panic("unreachable")
		}

		// initialize fields
		l.index = &Index{
			File: l.Root.IndexFile(l.ID),
		}
		if err = l.index.Open(); err != nil {
			err = fmt.Errorf("failed to open Index: File=%s err=%s", l.index.File, err)
			return
		}
		l.symbols = &logutil.Symbols{}
		l.symbols.Init()

		l.symbolsEditor = &logutil.SymbolsEditor{}
		l.symbolsEditor.Init(l.symbols)

		if status == LogCreated {
			// load Index
			if err = l.index.Load(); err != nil {
				err = fmt.Errorf("failed to load Index: File=%s err=%s", l.index.File, err)
				return
			}

			// load Symbols
			symbolsReader := &SymbolsReader{
				File: l.Root.SymbolFile(l.ID),
				SymbolsEditor: &logutil.SymbolsEditor{
					KeepID: true,
				},
			}
			if err = symbolsReader.Open(); err != nil {
				return
			}
			symbolsReader.SymbolsEditor.Init(l.symbols)
			if err = symbolsReader.Load(); err != nil {
				err = fmt.Errorf("failed to load Symbols: File=%s err=%s", symbolsReader.File, err)
				return
			}
			if err = symbolsReader.Close(); err != nil {
				err = fmt.Errorf("failed to close Symbols: File=%s err=%s", symbolsReader.File, err)
				return
			}
		}
	})
	return
}

// LogReaderを返す。
func (l *Log) Reader() (*LogReader, error) {
	if err := l.load(); err != nil {
		return nil, err
	}

	l.lock.Lock()
	defer l.lock.Unlock()
	return newLogReader(l)
}

// LogWriterを返す。この関数は、常に同じLogWriterのインスタンスを返す。
// なぜなら、1つのログに対してLogWriterは1つまでしか作ることができないから。
func (l *Log) Writer() (*LogWriter, error) {
	if err := l.load(); err != nil {
		return nil, err
	}

	l.lock.Lock()
	defer l.lock.Unlock()
	if l.w == nil {
		// create new writer
		w, err := newLogWriter(l)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize LogWriter(%s): %s", l.ID, err.Error())
		}
		l.w = w
	}
	return l.w, nil
}

// Logの状態を確認する。
func (l *Log) Status() LogStatus {
	m := l.Root.MetaFile(l.ID).Exists()
	r := l.Root.RawFuncLogFile(l.ID, 0).Exists()
	i := l.Root.IndexFile(l.ID).Exists()
	s := l.Root.SymbolFile(l.ID).Exists()

	if m && r && i && s {
		return LogCreated
	} else if !m && !r && !i && !s {
		return LogNotCreated
	} else {
		return LogBroken
	}
}

// ログファイルを削除する。
// すべてのLogReaderとLogWriterをCloseした後に呼び出すこと。
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

func newLogReader(l *Log) (*LogReader, error) {
	r := &LogReader{
		l: l,
	}
	if err := r.init(); err != nil {
		return nil, err
	}
	return r, nil
}
func (lr *LogReader) init() error {
	return nil
}
func (lr *LogReader) Close() error {
	return nil
}

// 指定した期間のRawFuncLogを返す。
// この操作を実行中、他の操作はブロックされる。
func (lr *LogReader) Search(start, end time.Time, fn func(evt logutil.RawFuncLogNew) error) error {
	lr.l.lock.RLock()
	defer lr.l.lock.RUnlock()

	var startIdx int64
	var endIdx int64

	if err := lr.l.index.Walk(func(i int64, ir IndexRecord) error {
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

	for i := startIdx; i <= endIdx; i++ {
		return lr.walkRawFuncLogFile(i, fn)
	}
	return nil
}

// Symbolsを返す。
func (lr *LogReader) Symbols() *logutil.Symbols {
	return lr.l.symbols
}

// 関数呼び出しのログを先頭からすべて読み込む。
// この操作を実行中、他の操作はブロックされる。
func (lr *LogReader) Walk(fn func(evt logutil.RawFuncLogNew) error) error {
	lr.l.lock.RLock()
	defer lr.l.lock.RUnlock()

	return lr.l.index.Walk(func(i int64, _ IndexRecord) error {
		return lr.walkRawFuncLogFile(i, fn)
	})
}

func (lr *LogReader) walkRawFuncLogFile(i int64, fn func(evt logutil.RawFuncLogNew) error) error {
	if lr.l.index.Get(i).IsWriting() {
		// 書き込み中のファイルからすべてのレコードを読み出すことはできない。
		// そのため、キャッシュを使用する。
		for _, evt := range lr.l.lastFuncLogFileCache {
			if err := fn(*evt); err != nil {
				return err
			}
		}
		return nil
	} else {
		// 通常通りファイルからレコードを読み出す。
		fl := RawFuncLogReader{
			File: lr.l.Root.RawFuncLogFile(lr.l.ID, i),
		}
		if err := fl.Open(); err != nil {
			return err
		}
		if err := fl.Walk(fn); err != nil {
			fl.Close() // nolinter: errchk
			return err
		}
		if err := fl.Close(); err != nil {
			return err
		}
		return nil
	}
}

func newLogWriter(l *Log) (*LogWriter, error) {
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
	// NOTE: init()の呼び出し元はnewLogWriter()である。
	//       newLogWriter()の呼び出し元はLog.Writer()であり、そこでロックをかけている。
	//       そのため、ここでロックをかけてはいけない。
	var err error
	checkError := func(errprefix string, e error) {
		if e != nil && err == nil {
			err = errors.New(fmt.Sprintf("%s: %s", errprefix, e.Error()))
		}
	}

	checkError("failed open lasat func log file", lw.openRawFuncLog())

	lw.symbolsWriter = &SymbolsWriter{File: lw.l.Root.SymbolFile(lw.l.ID)}
	checkError("failed open symbolsWriter file", lw.symbolsWriter.Open())
	return err
}

func (lw *LogWriter) Close() error {
	var err error
	checkError := func(logprefix string, e error) {
		if e != nil && e == nil {
			err = fmt.Errorf("%s: %s", logprefix, e.Error())
		}
	}

	lw.l.lock.Lock()
	defer lw.l.lock.Unlock()
	w, err := lw.l.Root.MetaFile(lw.l.ID).OpenWriteOnly()
	if err != nil {
		return errors.New("can not open meta data file: " + err.Error())
	}
	defer w.Close() // nolint: errcheck
	if err := json.NewEncoder(w).Encode(lw.l.Metadata); err != nil {
		return errors.New("can not write meta data file: " + err.Error())
	}

	checkError("failed close last func log file", lw.closeRawFuncLog())
	checkError("failed close symbolsWriter file", lw.symbolsWriter.Close())
	log.Println("INFO: storage logs closed")
	return err
}

func (lw *LogWriter) AppendFuncLog(raw *logutil.RawFuncLogNew) error {
	lw.l.lock.Lock()
	defer lw.l.lock.Unlock()

	if err := lw.autoRotate(); err != nil {
		return err
	}
	if err := lw.funcLogWriter.Append(raw); err != nil {
		return err
	}

	// update IndexRecord
	lw.lastIndexRecord.Records++
	lw.lastIndexRecord.Timestamp = time.Unix(raw.Timestamp, 0)
	if err := lw.l.index.UpdateLast(lw.lastIndexRecord); err != nil {
		return err
	}

	// update shared FuncLogFile cache
	lw.l.lastFuncLogFileCache = append(lw.l.lastFuncLogFileCache, raw)
	return nil
}

func (lw *LogWriter) AppendSymbols(symbols *logutil.Symbols) error {
	lw.l.lock.Lock()
	defer lw.l.lock.Unlock()

	if err := lw.symbolsWriter.Append(symbols); err != nil {
		return err
	}
	lw.l.symbolsEditor.AddSymbols(symbols)
	return nil
}

func (lw *LogWriter) Symbols() *logutil.Symbols {
	return lw.l.symbols
}

// RawFuncLogファイルんサイズがMaxFileSizeよりも大きい場合、ファイルのローテーションを行う。。
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

// RawFuncLogファイルのローテーションを行う。
// callee MUST call "l.lock.Lock()" before call l.autoRotate().
func (lw *LogWriter) rotate() error {
	if err := lw.closeRawFuncLog(); err != nil {
		return fmt.Errorf("failed rotate: %s", err)
	}
	if err := lw.openRawFuncLog(); err != nil {
		return fmt.Errorf("failed rotate: %s", err)
	}
	return nil
}

// 新しいRawFuncLogFileを作り、開く。
func (lw *LogWriter) openRawFuncLog() error {
	funcLogN := lw.l.index.Len()
	lw.funcLogWriter = &RawFuncLogWriter{File: lw.l.Root.RawFuncLogFile(lw.l.ID, funcLogN)}
	if err := lw.funcLogWriter.Open(); err != nil {
		return fmt.Errorf("cannot open FuncLogWriter(file=%s): %s", lw.funcLogWriter.File, err)
	}

	lw.lastIndexRecord = IndexRecord{
		Timestamp: time.Unix(0, 0),
		Records:   0,
	}
	lw.lastIndexRecord.writing = true
	if err := lw.l.index.Append(lw.lastIndexRecord); err != nil {
		return fmt.Errorf("cannot append a IndexRecord: %s", err)
	}
	return nil
}

// 現在開いているRawFuncLogFileを閉じる。
func (lw *LogWriter) closeRawFuncLog() error {
	if err := lw.funcLogWriter.Close(); err != nil {
		return fmt.Errorf("cannot close FuncLogWriter: %s", err)
	}
	lw.lastIndexRecord.writing = false
	if err := lw.l.index.UpdateLast(lw.lastIndexRecord); err != nil {
		return fmt.Errorf("cannot write index record: %s", err)
	}

	// RawFuncLogFileのキャッシュをクリア
	lw.l.lastFuncLogFileCache = make([]*logutil.RawFuncLogNew, 0)
	return nil
}

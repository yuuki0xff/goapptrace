package storage

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

type LogID [16]byte

// 指定したLogIDに対応するログの作成・読み書き・削除を行う。
// 現時点では完全なスレッドセーフではない。読み書きを複数のgoroutineから同時に行うのは推薦しない。
// Open()するとMaxFileSize程度のオンメモリキャッシュが確保される。メモリ使用量に注意。
//
// ログは下記の5つから構成されている。RawFuncLogに関しては、MaxFileSizeに収まるようにファイルをローテーションされる。
//  * LogMetadata
//  * FuncLogRawFuncLog
//  * Symbolキャッシュ
//  * Index
type Log struct {
	ID LogID
	// メタデータを更新するたびにインクリメントされる値
	Version  int
	Root     DirLayout
	Metadata *LogMetadata
	// 書き込み先ファイルが変更される直前に呼び出される。
	// このイベント実行中はロックが外れるため、他のスレッドから随時書き込まれる可能性がある。
	BeforeRotateEventHandler func()
	// RawFuncLogファイルの最大のファイルサイズの目安。
	// 実際のファイルサイズは、指定したサイズよりもやや大きくなる可能性がある。
	// 0を指定するとローテーション機能が無効になる。
	MaxFileSize int64
	ReadOnly    bool

	lock sync.RWMutex
	// rotate()を実行中ならtrue。
	// rotate()内部で発生するBeforeRotate eventの実行中は、ロックを外さなければならない。
	// そのイベント実行中に、並行してrotate()が実行されないように排他制御するためのフラグ。
	rotating bool
	// フィアルがcloseされていたらtrue。
	// trueなら全ての操作を受け付けてはならない。
	closed bool

	index         *Index
	symbols       *logutil.Symbols
	symbolsEditor *logutil.SymbolsEditor // readonlyならnil
	symbolsWriter *SymbolsWriter         // readonlyならnil

	funcLog      SplitReadWriter
	rawFuncLog   SplitReadWriter
	goroutineLog SplitReadWriter
}

// Logオブジェクトをmarshalするときに使用する。
// Logとは異なる点は、APIのレスポンスに必要なフィールドしか持っていないこと、および
// フィールドの値が更新されないため、ロックセずにフィールドの値にアクセスできることである。
// APIのレスポンスとして使用することを想定している。
type LogInfo struct {
	ID          string      `json:"log-id"`
	Version     int         `json:"version"`
	Metadata    LogMetadata `json:"metadata"`
	MaxFileSize int64       `json:"max-file-size"`
	ReadOnly    bool        `json:"read-only"`
}
type LogMetadata struct {
	// Timestamp of the last record
	Timestamp time.Time `json:"timestamp"`

	// The configuration of user interface
	UI UIConfig `json:"ui"`
}

type UIConfig struct {
	FuncCalls  map[logutil.FuncLogID]UIItemConfig `json:"func-calls"`
	Funcs      map[logutil.FuncID]UIItemConfig    `json:"funcs"`
	Goroutines map[logutil.GID]UIItemConfig       `json:"goroutines"`
}
type UIItemConfig struct {
	Pinned  bool   `json:"pinned"`
	Masked  bool   `json:"masked"`
	Comment string `json:"comment"`
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
	ErrConflict   = errors.New("failed to update because conflict")
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
		err = fmt.Errorf(
			"missmatch id length. expect %d charactors, but %d",
			2*len(id), 2*len(buf),
		)
		return
	}
	copy(id[:], buf)
	return
}

// 16進数化した文字列として出力する。
func (id LogID) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('"')
	buf.Write([]byte(id.Hex()))
	buf.WriteByte('"')
	return buf.Bytes(), nil
}

// 16進数値のような文字列からLogIDに変換する。
func (id *LogID) UnmarshalJSON(data []byte) error {
	if len(data) != len(id)*2+2 {
		return errors.New("mismatch id length")
	}

	if data[0] != '"' || data[len(data)-1] != '"' {
		return errors.New("missing '\"'")
	}

	newId, err := id.Unhex(string(data[1 : len(data)-1]))
	if err != nil {
		return err
	}
	*id = newId
	return nil
}

// LogIDを16進数表現で返す。
func (id LogID) String() string {
	return id.Hex()
}

// このログを開く。読み書きが可能。
// Openするとファイルが作成されるため、LogStatusがLogInitializedに変化する。
func (l *Log) Open() error {
	l.lock.Lock()
	defer l.lock.Unlock()

	// initialize Version
	if l.Version < 1 {
		l.Version = 1
	}

	// initialize Metadata
	if l.Metadata == nil {
		l.Metadata = &LogMetadata{}
		metaFile := l.Root.MetaFile(l.ID)
		if metaFile.Exists() {
			// load metadata
			r, err := metaFile.OpenReadOnly()
			if err != nil {
				return fmt.Errorf("failed to open metadata file: %s", err.Error())
			}
			defer r.Close() // nolint: errcheck
			if err := json.NewDecoder(r).Decode(l.Metadata); err != nil {
				return fmt.Errorf("failed to read metadata file: %s", err.Error())
			}
		}
	}

	// check Log file status
	status := l.Status()
	switch status {
	case LogBroken:
		return fmt.Errorf("Log(%s) is broken", l.ID)
	case LogNotCreated:
	case LogCreated:
		break
	default:
		log.Panicf("bug: unexpected status: status=%+v", status)
		panic("unreachable")
	}

	// initialize fields
	l.closed = false
	l.index = &Index{
		File:     l.Root.IndexFile(l.ID),
		ReadOnly: l.ReadOnly,
	}
	if err := l.index.Open(); err != nil {
		return fmt.Errorf("failed to open Index: File=%s err=%s", l.index.File, err)
	}
	l.symbols = &logutil.Symbols{}
	l.symbols.Init()
	if !l.ReadOnly {
		l.symbolsEditor = &logutil.SymbolsEditor{}
		l.symbolsEditor.Init(l.symbols)
		l.symbolsWriter = &SymbolsWriter{
			File: l.Root.SymbolFile(l.ID),
		}
		if err := l.symbolsWriter.Open(); err != nil {
			return fmt.Errorf("failed to open SymbolsWriter: File=%s err=%s", l.symbolsWriter.File, err)
		}
	}
	if status == LogCreated {
		// load Index
		if err := l.index.Load(); err != nil {
			return fmt.Errorf("failed to load Index: File=%s err=%s", l.index.File, err)
		}

		// load Symbols
		symbolsReader := &SymbolsReader{
			File: l.Root.SymbolFile(l.ID),
			SymbolsEditor: &logutil.SymbolsEditor{
				KeepID: true,
			},
		}
		if err := symbolsReader.Open(); err != nil {
			return fmt.Errorf("failed to load Symbols: File=%s err=%s", symbolsReader.File, err)
		}
		symbolsReader.SymbolsEditor.Init(l.symbols)
		if err := symbolsReader.Load(); err != nil {
			return fmt.Errorf("failed to load Symbols: File=%s err=%s", symbolsReader.File, err)
		}
		if err := symbolsReader.Close(); err != nil {
			return fmt.Errorf("failed to close Symbols: File=%s err=%s", symbolsReader.File, err)
		}
	}

	// open log files
	l.funcLog = SplitReadWriter{
		FileNamePattern: func(index int) File {
			return l.Root.FuncLogFile(l.ID, int64(index))
		},
		ReadOnly: l.ReadOnly,
	}
	l.rawFuncLog = SplitReadWriter{
		FileNamePattern: func(index int) File {
			return l.Root.RawFuncLogFile(l.ID, int64(index))
		},
		ReadOnly: l.ReadOnly,
	}
	l.goroutineLog = SplitReadWriter{
		FileNamePattern: func(index int) File {
			return l.Root.GoroutineLogFile(l.ID, int64(index))
		},
		ReadOnly: l.ReadOnly,
	}

	if err := l.funcLog.Open(); err != nil {
		return err
	}
	if err := l.rawFuncLog.Open(); err != nil {
		return err
	}
	if err := l.goroutineLog.Open(); err != nil {
		return err
	}
	return nil
}

func (l *Log) Close() error {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.closed = true

	if err := l.index.Close(); err != nil {
		return err
	}
	// 書き込み可能ならClose()する。
	// 読み込み専用のときは、l.symbolsWriter==nilなのでClose()しない。
	if !l.ReadOnly {
		if err := l.symbolsWriter.Close(); err != nil {
			return err
		}
	}
	if err := l.funcLog.Close(); err != nil {
		return err
	}
	if err := l.rawFuncLog.Close(); err != nil {
		return err
	}
	if err := l.goroutineLog.Close(); err != nil {
		return err
	}

	// write MetaData
	w, err := l.Root.MetaFile(l.ID).OpenWriteOnly()
	if err != nil {
		return errors.New("can not open meta data file: " + err.Error())
	}
	if err := json.NewEncoder(w).Encode(l.Metadata); err != nil {
		w.Close() // nolint: errcheck
		return errors.New("can not write meta data file: " + err.Error())
	}
	return w.Close()
}

// Logの状態を確認する。
func (l *Log) Status() LogStatus {
	m := l.Root.MetaFile(l.ID).Exists()
	f := l.Root.FuncLogFile(l.ID, 0).Exists()
	r := l.Root.RawFuncLogFile(l.ID, 0).Exists()
	i := l.Root.IndexFile(l.ID).Exists()
	s := l.Root.SymbolFile(l.ID).Exists()

	if m && f && r && i && s {
		return LogCreated
	} else if !m && !f && !r && !i && !s {
		return LogNotCreated
	} else {
		return LogBroken
	}
}

// ログファイルを削除する。
// すべてのLogとLogをCloseした後に呼び出すこと。
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

		file = l.Root.FuncLogFile(l.ID, index)
		if err := file.Remove(); err != nil {
			return fmt.Errorf("failed to remove the FuncLog(%s): %s", l.ID, err.Error())
		}

		index++
	}
	if err := l.Root.SymbolFile(l.ID).Remove(); err != nil {
		return fmt.Errorf("failed to remove the Symbol(%s): %s", l.ID, err.Error())
	}
	return nil
}

// 指定した期間のRawFuncLogを返す。
// この操作を実行中、他の操作はブロックされる。
func (l *Log) Search(start, end time.Time, fn func(evt logutil.RawFuncLog) error) error {
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

	for i := startIdx; i <= endIdx; i++ {
		return l.WalkRawFuncLogFile(i, fn)
	}
	return nil
}

// 関数呼び出しに関するログを先頭から全て読み込む。
// この操作を実行中、他の操作はブロックされる
func (l *Log) WalkFuncLog(fn func(evt logutil.FuncLog) error) error {
	l.lock.RLock()
	size := l.index.Len()
	l.lock.RUnlock()

	for i := int64(0); i < size; i++ {
		if err := l.WalkFuncLogFile(i, fn); err != nil {
			return err
		}
	}
	return nil
}

// 指定したindexのファイルの内容を全てcallbackする
func (l *Log) WalkFuncLogFile(i int64, fn func(evt logutil.FuncLog) error) error {
	// SplitReadWriterのIndex()やWalk()は排他制御されているため、、
	// ここでl.lock.RLock()をする必要がない。
	return l.funcLog.Index(int(i)).Walk(
		func() interface{} {
			return &logutil.FuncLog{}
		},
		func(val interface{}) error {
			data := val.(*logutil.FuncLog)
			return fn(*data)
		},
	)
}

// 関数呼び出しのログを先頭からすべて読み込む。
// この操作を実行中、他の操作はブロックされる。
func (l *Log) WalkRawFuncLog(fn func(evt logutil.RawFuncLog) error) error {
	l.lock.RLock()
	size := l.index.Len()
	l.lock.RUnlock()

	for i := int64(0); i < size; i++ {
		if err := l.WalkRawFuncLogFile(i, fn); err != nil {
			return err
		}
	}
	return nil
}

// 指定したindexのファイルの内容を全てcallbackする
func (l *Log) WalkRawFuncLogFile(i int64, fn func(evt logutil.RawFuncLog) error) error {
	// SplitReadWriterのIndex()やWalk()は排他制御されているため、、
	// ここでl.lock.RLock()をする必要がない。
	return l.rawFuncLog.Index(int(i)).Walk(
		func() interface{} {
			return &logutil.RawFuncLog{}
		},
		func(val interface{}) error {
			data := val.(*logutil.RawFuncLog)
			return fn(*data)
		},
	)
}

// TODO: テストを書く
// 指定したindexの範囲で活動していたgoroutineを全てcallbackする。
func (l *Log) WalkGoroutine(i int64, fn func(g logutil.Goroutine) error) error {
	return l.goroutineLog.Index(int(i)).Walk(
		func() interface{} {
			return &logutil.Goroutine{}
		},
		func(val interface{}) error {
			data := val.(*logutil.Goroutine)
			return fn(*data)
		},
	)
}

// IndexRecordの内容を全てcallbackする。
func (l *Log) WalkIndexRecord(fn func(i int64, ir IndexRecord) error) error {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.index.Walk(fn)
}
func (l *Log) IndexLen() int64 {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.index.Len()
}

// FuncLogを追加する。
// ファイルが閉じられていた場合、os.ErrClosedを返す。
func (l *Log) AppendFuncLog(funcLog *logutil.FuncLog) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.closed {
		return os.ErrClosed
	}

	return l.funcLog.Append(funcLog)
}

// RawFuncLogを追加する。
// ファイルサイズが上限に達していた場合、ファイルを分割する。
// ファイルが閉じられていた場合、os.ErrClosedを返す。
func (l *Log) AppendRawFuncLog(raw *logutil.RawFuncLog) error {
	if l.closed {
		return os.ErrClosed
	}

	if err := l.autoRotate(); err != nil {
		return err
	}

	l.lock.Lock()
	defer l.lock.Unlock()
	if err := l.rawFuncLog.Append(raw); err != nil {
		return err
	}

	// update IndexRecord
	if l.index.Len() > 0 {
		last := l.index.Last()
		last.Records++
		last.Timestamp = time.Unix(raw.Timestamp, 0)
		if err := l.index.UpdateLast(last); err != nil {
			return err
		}
	} else {
		if err := l.index.Append(IndexRecord{
			Timestamp: time.Unix(raw.Timestamp, 0),
			Records:   1,
			writing:   true,
		}); err != nil {
			return err
		}
	}
	return nil
}

// Symbolsを書き込む。
func (l *Log) AppendSymbols(symbols *logutil.Symbols) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.closed {
		return os.ErrClosed
	}

	if err := l.symbolsWriter.Append(symbols); err != nil {
		return err
	}
	l.symbolsEditor.AddSymbols(symbols)
	return nil
}

// Goroutineのステータスを書き込む
func (l *Log) AppendGoroutine(g *logutil.Goroutine) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.closed {
		return os.ErrClosed
	}

	return l.goroutineLog.Append(g)
}

func (l *Log) Symbols() *logutil.Symbols {
	return l.symbols
}

// TODO: テストを書く
// Metadataフィールドを更新して、Versionをインクリメントする。
func (l *Log) UpdateMetadata(currentVer int, metadata *LogMetadata) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.Version != currentVer {
		return ErrConflict
	}
	l.Version++
	l.Metadata = metadata
	return nil
}

// RawFuncLogファイルサイズがMaxFileSizeよりも大きい場合、ファイルのローテーションを行う。。
// 呼び出し元でロックを取得していてはいけない。
func (l *Log) autoRotate() error {
	l.lock.RLock()
	unlockOnce := sync.Once{}
	unlock := func() {
		unlockOnce.Do(l.lock.RUnlock)
	}
	defer unlock()

	if l.rawFuncLog.SplitCount() == 0 {
		// まだファイルが存在しないので、rotateの必要なし
		return nil
	}

	last, err := l.rawFuncLog.LastFile()
	if err != nil {
		return err
	}
	size, err := last.File.Size()
	if err != nil {
		return err
	}
	if l.MaxFileSize != 0 && size > l.MaxFileSize {
		// l.rotate()を呼び出す前に、ロックを解除しなければならない。
		unlock()
		return l.rotate()
	}
	return nil
}

// 書き込み先ファイルのローテーションを行う。
// 並行して実行しているrotate()が存在することを検出した場合、ローテーションが中断される。
// 実行中には、BeforeRotateイベントが発生する。
// 呼び出し元でロックを取得していてはいけない。
func (l *Log) rotate() error {
	l.lock.Lock()
	if l.rotating {
		// 他のgoroutineでrotate()が実行中だったため、ローテーションをしない。
		l.lock.Unlock()
		return nil
	}
	l.rotating = true
	l.lock.Unlock()

	l.raiseBeforeRotateEvent()

	l.lock.Lock()
	defer l.lock.Unlock()
	// Unlockする前にrotatingフラグを元に戻す。
	defer func() {
		l.rotating = false
	}()

	if err := l.funcLog.Rotate(); err != nil {
		return err
	}
	if err := l.rawFuncLog.Rotate(); err != nil {
		return err
	}
	if err := l.goroutineLog.Rotate(); err != nil {
		return err
	}

	if l.index.Len() > 0 {
		last := l.index.Last()
		last.writing = false
		if err := l.index.UpdateLast(last); err != nil {
			return err
		}
	}

	return l.index.Append(IndexRecord{
		Timestamp: time.Unix(0, 0),
		Records:   0,
		writing:   true,
	})
}

// BeforeRotateEventHandlerを呼び出す。
func (l *Log) raiseBeforeRotateEvent() {
	if l.BeforeRotateEventHandler != nil {
		l.BeforeRotateEventHandler()
	}
}

// ロックをかけた上で、JSONに変換する
func (l *Log) LogInfo() LogInfo {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return LogInfo{
		ID:          l.ID.Hex(),
		Version:     l.Version,
		Metadata:    *l.Metadata,
		MaxFileSize: l.MaxFileSize,
		ReadOnly:    l.ReadOnly,
	}
}

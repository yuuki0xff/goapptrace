package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

type LogID = types.LogID

const (
	defaultRotateInterval         = 100000
	defaultMetadataUpdateInterval = 1 * time.Second
)

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
	Metadata *types.LogMetadata
	// 書き込み先ファイルが変更される直前に呼び出される。
	// このイベント実行中はロックが外れるため、他のスレッドから随時書き込まれる可能性がある。
	BeforeRotateEventHandler func()
	// RawFuncLogファイルの最大のファイルサイズの目安。
	// 実際のファイルサイズは、指定したサイズよりもやや大きくなる可能性がある。
	// 0を指定するとローテーション機能が無効になる。
	MaxFileSize int64
	ReadOnly    bool

	lock   sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	// フィアルがcloseされていたらtrue。
	// trueなら全ての操作を受け付けてはならない。
	closed bool
	// rotate()を実行中ならtrue。
	// rotate()内部で発生するBeforeRotate eventの実行中は、ロックを外さなければならない。
	// そのイベント実行中に、並行してrotate()が実行されないように排他制御するためのフラグ。
	rotating bool
	// autoRotate()の呼び出しを連続してスキップした回数。
	// この変数は、AppendRawFuncLog()を呼び出すたびにincrementされる。
	// 値がrotateIntervalより大きくなったときは、値が0に戻り、autoRotate()が実行される。
	autorotateSkips int
	// autoRotate()を呼び出す間隔。詳細はautorotateSkipsのドキュメントを参照すること。
	rotateInterval int

	index   *Index
	symbols *types.Symbols

	funcLog      SplitReadWriter
	rawFuncLog   SplitReadWriter
	goroutineLog SplitReadWriter
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
		l.Metadata = &types.LogMetadata{}
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
	l.ctx, l.cancel = context.WithCancel(context.Background())
	l.closed = false
	if l.rotateInterval == 0 {
		l.rotateInterval = defaultRotateInterval
	}
	l.index = &Index{
		File:     l.Root.IndexFile(l.ID),
		ReadOnly: l.ReadOnly,
	}
	if err := l.index.Open(); err != nil {
		return fmt.Errorf("failed to open Index: File=%s err=%s", l.index.File, err)
	}
	l.symbols = &types.Symbols{
		Writable: !l.ReadOnly,
	}
	l.symbols.Init()
	if status == LogCreated {
		// load Index
		if err := l.index.Load(); err != nil {
			return fmt.Errorf("failed to load Index: File=%s err=%s", l.index.File, err)
		}

		// load Symbols
		if err := l.symbolsStore().Read(l.symbols); err != nil {
			return errors.Wrap(err, "failed to load symbols")
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

	if !l.ReadOnly {
		// 買い込み可能なので、定期的にMetadataのタイムスタンプを更新する必要がある。。
		l.wg.Add(1)
		go l.timestampUpdateWorker()
	}
	return nil
}

func (l *Log) Close() error {
	l.lock.Lock()
	l.cancel()
	l.lock.Unlock()

	l.wg.Wait()

	l.lock.Lock()
	defer l.lock.Unlock()
	l.closed = true

	if err := l.index.Close(); err != nil {
		return err
	}
	// 書き込み可能ならClose()する。
	// 読み込み専用のときは、l.symbolsWriter==nilなのでClose()しない。
	if !l.ReadOnly {
		if err := l.symbolsStore().Write(l.symbols); err != nil {
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
func (l *Log) Search(start, end time.Time, fn func(evt types.RawFuncLog) error) error {
	l.lock.RLock()
	defer l.lock.RUnlock()

	var startIdx int64
	var endIdx int64

	if err := l.index.Walk(func(i int64, ir IndexRecord) error {
		if start.Before(ir.Timestamp.UnixTime()) {
			startIdx = i - 1
		} else if end.Before(ir.Timestamp.UnixTime()) {
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
func (l *Log) WalkFuncLog(fn func(evt types.FuncLog) error) error {
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
func (l *Log) WalkFuncLogFile(i int64, fn func(evt types.FuncLog) error) error {
	// SplitReadWriterのIndex()やWalk()は排他制御されているため、、
	// ここでl.lock.RLock()をする必要がない。
	return l.funcLog.Index(int(i)).Walk(
		func() interface{} {
			return &types.FuncLog{}
		},
		func(val interface{}) error {
			data := val.(*types.FuncLog)
			return fn(*data)
		},
	)
}

// 関数呼び出しのログを先頭からすべて読み込む。
// この操作を実行中、他の操作はブロックされる。
func (l *Log) WalkRawFuncLog(fn func(evt types.RawFuncLog) error) error {
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
func (l *Log) WalkRawFuncLogFile(i int64, fn func(evt types.RawFuncLog) error) error {
	// SplitReadWriterのIndex()やWalk()は排他制御されているため、、
	// ここでl.lock.RLock()をする必要がない。
	return l.rawFuncLog.Index(int(i)).Walk(
		func() interface{} {
			return &types.RawFuncLog{}
		},
		func(val interface{}) error {
			data := val.(*types.RawFuncLog)
			return fn(*data)
		},
	)
}

// TODO: テストを書く
// 指定したindexの範囲で活動していたgoroutineを全てcallbackする。
func (l *Log) WalkGoroutine(i int64, fn func(g types.Goroutine) error) error {
	return l.goroutineLog.Index(int(i)).Walk(
		func() interface{} {
			return &types.Goroutine{}
		},
		func(val interface{}) error {
			data := val.(*types.Goroutine)
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
func (l *Log) AppendFuncLog(funcLog *types.FuncLog) error {
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
func (l *Log) AppendRawFuncLog(raw *types.RawFuncLog) error {
	if l.closed {
		return os.ErrClosed
	}

	// AppendRawFuncLog()を高速化するために、autoRotate()の実行回数を減らす。
	l.autorotateSkips++
	if l.autorotateSkips > l.rotateInterval {
		// *SLOW PATH*
		l.autorotateSkips = 0

		// ファイルの自動ローテーションするか否かをチェックするためにファイルサイズを参照する。
		// そのファイルサイズの参照が意外と重たい。。。
		if err := l.autoRotate(); err != nil {
			return err
		}
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
		last.Timestamp = raw.Timestamp
		if err := l.index.UpdateLast(last); err != nil {
			return err
		}
	} else {
		if err := l.index.Append(IndexRecord{
			Timestamp: raw.Timestamp,
			Records:   1,
			writing:   true,
		}); err != nil {
			return err
		}
	}
	return nil
}

// Symbolsにセットする
func (l *Log) SetSymbolsData(data *types.SymbolsData) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.closed {
		return os.ErrClosed
	}

	l.symbols.Load(*data)
	return nil
}

// Goroutineのステータスを書き込む
func (l *Log) AppendGoroutine(g *types.Goroutine) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.closed {
		return os.ErrClosed
	}

	return l.goroutineLog.Append(g)
}

func (l *Log) Symbols() *types.Symbols {
	return l.symbols
}

// TODO: テストを書く
// Metadataフィールドを更新して、Versionをインクリメントする。
func (l *Log) UpdateMetadata(currentVer int, metadata *types.LogMetadata) error {
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
		writing: true,
	})
}

// BeforeRotateEventHandlerを呼び出す。
func (l *Log) raiseBeforeRotateEvent() {
	if l.BeforeRotateEventHandler != nil {
		l.BeforeRotateEventHandler()
	}
}

// ロックをかけた上で、JSONに変換する
func (l *Log) LogInfo() types.LogInfo {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return types.LogInfo{
		ID:          l.ID.Hex(),
		Version:     l.Version,
		Metadata:    *l.Metadata,
		MaxFileSize: l.MaxFileSize,
		ReadOnly:    l.ReadOnly,
	}
}

// タイムスタンプを定期的に更新する
func (l *Log) timestampUpdateWorker() {
	defer l.wg.Done()
	ticker := time.NewTicker(defaultMetadataUpdateInterval)
	defer ticker.Stop()

	update := func() {
		var needUpdate bool
		var meta *types.LogMetadata
		var ver int
		func() {
			l.lock.RLock()
			defer l.lock.RUnlock()

			if l.index.Len() == 0 {
				return
			}
			ir := l.index.Last()
			if ir.Timestamp.UnixTime() == l.Metadata.Timestamp {
				return
			}
			needUpdate = true
			meta = &types.LogMetadata{}
			*meta = *l.Metadata
			meta.Timestamp = ir.Timestamp.UnixTime()
			ver = l.Version
		}()
		if needUpdate {
			if err := l.UpdateMetadata(ver, meta); err != nil {
				log.Println(errors.Wrap(err, "can not update timestamp"))
			}
		}
	}

	for {
		select {
		case <-l.ctx.Done():
			update()
			return
		case <-ticker.C:
			update()
		}
	}
}

func (l *Log) symbolsStore() SymbolsStore {
	return SymbolsStore{
		File:     l.Root.SymbolFile(l.ID),
		ReadOnly: l.ReadOnly,
	}
}

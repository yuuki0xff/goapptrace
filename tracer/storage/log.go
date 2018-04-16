package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/tracer/encoding"
	"github.com/yuuki0xff/goapptrace/tracer/types"
)

type LogID = types.LogID

const (
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

	index   *Index
	symbols *types.Symbols

	funcLog      FuncLogStore
	rawFuncLog   RawFuncLogStore
	goroutineLog GoroutineStore

	// LogInfoが更新されたことを通知する
	event logEvent
}

type logEvent struct {
	lock      sync.RWMutex
	callbacks map[int]func(info *types.LogInfo)
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
	l.funcLog = FuncLogStore{
		Store: Store{
			File:       l.Root.FuncLogFile(l.ID, 0),
			RecordSize: int(encoding.SizeFuncLog()),
			ReadOnly:   l.ReadOnly,
		},
	}
	l.rawFuncLog = RawFuncLogStore{
		Store: Store{
			File:       l.Root.RawFuncLogFile(l.ID, 0),
			RecordSize: int(encoding.SizeRawFuncLog()),
			ReadOnly:   l.ReadOnly,
		},
	}
	l.goroutineLog = GoroutineStore{
		Store: Store{
			File:       l.Root.GoroutineLogFile(l.ID, 0),
			RecordSize: int(encoding.SizeGoroutine()),
			ReadOnly:   l.ReadOnly,
		},
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
		// 書き込み可能なので、定期的にMetadataのタイムスタンプを更新する必要がある。。
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

// 指定した期間のFuncLogを返す。
// この操作を実行中、他の操作はブロックされる。
func (l *Log) SearchFuncLog(start, end types.Time, fn func(fl types.FuncLog) error) error {
	l.lock.RLock()
	defer l.lock.RUnlock()

	startIdx, endIdx := l.index.IDRangeByTime(start, end)
	if endIdx == 0 {
		endIdx = l.funcLog.Records()
	}

	l.funcLog.Lock()
	defer l.funcLog.Unlock()
	for i := startIdx; i < endIdx; i++ {
		var fl types.FuncLog
		err := l.funcLog.Get(types.FuncLogID(i), &fl)
		if err != nil {
			return err
		}
		err = fn(fl)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Log) FuncLog(fn func(store *FuncLogStore)) {
	l.funcLog.Lock()
	defer l.funcLog.Unlock()
	fn(&l.funcLog)
}

func (l *Log) RawFuncLog(fn func(store *RawFuncLogStore)) {
	l.rawFuncLog.Lock()
	defer l.rawFuncLog.Unlock()
	fn(&l.rawFuncLog)
}

func (l *Log) Goroutine(fn func(store *GoroutineStore)) {
	l.goroutineLog.Lock()
	defer l.goroutineLog.Unlock()
	fn(&l.goroutineLog)
}

func (l *Log) Index(fn func(index *Index)) {
	l.lock.RLock()
	defer l.lock.RUnlock()
	fn(l.index)
}

func (l *Log) IndexLen() int64 {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.index.Len()
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

	// calls event handlers.
	info := l.logInfoNolock()
	l.event.Notify(&info)
	return nil
}

// ロックをかけた上で、JSONに変換する
func (l *Log) LogInfo() types.LogInfo {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return l.logInfoNolock()
}
func (l *Log) logInfoNolock() types.LogInfo {
	return types.LogInfo{
		ID:          l.ID.Hex(),
		Version:     l.Version,
		Metadata:    *l.Metadata,
		MaxFileSize: l.MaxFileSize,
		ReadOnly:    l.ReadOnly,
	}
}

// LogInfoが更新されたときに、fnを呼び出す。
// fnの引数は、更新後のLogInfoが渡される。渡されたデータは書き換えてはいけない。
func (l *Log) Watch(ctx context.Context, fn func(info *types.LogInfo)) {
	l.event.Watch(ctx, fn)
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
			if ir.MaxEnd.UnixTime() == l.Metadata.Timestamp {
				return
			}
			needUpdate = true
			meta = &types.LogMetadata{}
			*meta = *l.Metadata
			meta.Timestamp = ir.MaxEnd.UnixTime()
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

// notify は、Watch()で登録されたコールバック関数を全て呼び出す。
// TracersStore のデータが更新されたときに必ず呼び出すこと。
func (le *logEvent) Notify(info *types.LogInfo) {
	le.lock.Lock()
	defer le.lock.Unlock()
	for _, fn := range le.callbacks {
		fn(info)
	}
}

// LogInfoが更新されたときにfnを呼び出す。
// ctxは
func (le *logEvent) Watch(ctx context.Context, fn func(info *types.LogInfo)) {
	key := le.register(fn)
	<-ctx.Done()
	le.unregister(key)
}

// fnをコールバック関数として登録して、キーを返す。
// keyはコールバック関数の登録解除するときに使用する。
func (le *logEvent) register(fn func(info *types.LogInfo)) int {
	le.lock.Lock()
	defer le.lock.Unlock()

	if le.callbacks == nil {
		le.callbacks = map[int]func(info *types.LogInfo){}
	}
	for {
		// callbacks のキーの重複を避けるために乱数を使用している。
		// 乱数の品質は問わない(脆弱でも構わない)ため、gasのwarningを無視する。
		key := rand.Int() // nolint: gas
		if _, ok := le.callbacks[key]; ok {
			continue
		}
		le.callbacks[key] = fn
		return key
	}
}

// keyに対応するコールバック関数の登録を解除する。
// これ以降は、指定したコールバック関数が呼び出されることはない。
// keyは、register()が返した値。
func (le *logEvent) unregister(key int) {
	le.lock.Lock()
	defer le.lock.Unlock()

	if le.callbacks == nil {
		return
	}
	delete(le.callbacks, key)
}

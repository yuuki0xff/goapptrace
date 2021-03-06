package logger

import (
	"errors"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/yuuki0xff/goapptrace/tracer/types"
)

const (
	defaultMaxRetry      = 50
	defaultRetryInterval = 1 * time.Second
	skips                = 3
)

var (
	ClosedError = errors.New("already closed")

	lock    = sync.Mutex{}
	symbols = types.Symbols{
		Writable: true,
	}
	initBuffer []*types.RawFuncLog
	sender     Sender

	// 通常の環境で実行したときは、gid()はこの関数が返した値を返す。
	// この変数が設定されていなければ、gid()はpanicする。
	// Unit test実行時にダミーのGIDを返す目的で使用する。
	dummyGid func() types.GID
)

func init() {
	// get all symbols in this process.
	var sd types.SymbolsData

	//@@GAT@useNonStandardRuntime@ /*
	/*/
	fname2fileID := func(fname string) types.FileID {
		for i, f := range sd.Files {
			if f == fname {
				return types.FileID(i)
			}
		}
		sd.Files = append(sd.Files, fname)
		return types.FileID(len(sd.Files) - 1)
	}
	runtime.IterateSymbols(
		func(minpc, maxpc uintptr, name string) {
			sd.Mods = append(sd.Mods, types.GoModule{
				Name:  name,
				MinPC: minpc,
				MaxPC: maxpc,
			})
		},
		func(pc uintptr, name string) {
			sd.Funcs = append(sd.Funcs, types.GoFunc{
				Entry: pc,
				Name:  name,
			})
		},
		func(pc uintptr, file string, line int32) {
			if line < 0 {
				log.Panicf("invalid line: pc=%d, file=%s, line=%d", pc, file, line)
			}

			sd.Lines = append(sd.Lines, types.GoLine{
				PC:     pc,
				FileID: fname2fileID(file),
				Line:   uint32(line),
			})
		},
	)
	//*/

	lock.Lock()
	// setup sender.
	setOutput()
	if sender == nil {
		log.Panicln("sender is nil")
	}

	// send SymbolsData
	if err := sender.SendSymbols(&sd); err != nil {
		log.Panic(err)
	}

	// send buffered logs on initBuffer.
	for _, raw := range initBuffer {
		if err := sender.SendLog(raw); err != nil {
			log.Panic(err)
		}
		types.RawFuncLogPool.Put(raw)
	}
	initBuffer = nil
	lock.Unlock()
	symbols.Init()
}

func gid() types.GID {
	// get GoroutineID (GID)
	//@@GAT@useNonStandardRuntime@ /*

	// ここはgoapptrace以外の環境でコンパイルしたときに実行される。
	if dummyGid == nil {
		// t通常の環境ではGIDを取得できないため、panicする。
		panic("not supported")
	} else {
		// 単体テストの実行時にはダミーのGIDを返す
		return dummyGid()
	}

	/*/

	// ここは、`goapptrace run`を用いてコンパイルしたときに実行される。
	// runtime.GoID()は、標準のruntimeパッケージ内に存在しない関数である。
	// tracer/builderパッケージによってパッチが当てられた環境でのみ使用可能。
	return types.GID(runtime.GoID())

	//*/
}

func sendLog(tag types.TagName, id types.TxID) {
	logmsg := types.RawFuncLogPool.Get().(*types.RawFuncLog)
	logmsg.ID = types.NewRawFuncLogID()
	logmsg.Tag = tag
	logmsg.Timestamp = types.NewTime(time.Now())
	// logmsg.Frames のバッファは既に確保されているため、それを再利用する。
	logmsg.GID = gid()
	logmsg.TxID = id

	// types.MaxStackSize を超えている場合、正しいログが取得できない。
	// スライスの長さが小さくされている可能性があるため、事前に限界まで拡張する。
	pclen := runtime.Callers(skips, logmsg.Frames[:cap(logmsg.Frames)])
	logmsg.Frames = logmsg.Frames[:pclen]

	// TODO: インライン化やループ展開により、正しくないデータが帰ってくる可能性がある問題を修正する。
	// これらは過去のコードであるが、今後の実装の参考になる可能性があるため、残しておく。
	//
	// symbolsに必要なシンボルを追加とlogmsg.Framesの作成を行う。
	//if useCallersFrames {
	//	// runtime.CallersFrames()を使用する。
	//	// インライン化やループ展開がされた場合に、行番号や呼び出し元関数の調整を行うことができる。
	//	// しかし、オーバーヘッドが大きくなる。
	//	// コンパイラの最適化を簡単に無効化できない場合に使用することを推薦する。
	//} else {
	//	// runtime.FuncForPC()を使用する。
	//	// runtime.CallersFrames()を使用するよりもオーバーヘッドが少ない。
	//	// ただし、最適化が行われると呼び出し元の判定が狂ってしまう。
	//	// これを使用するときは、*最適化を無効*にしてコンパイルすること。
	//}

	lock.Lock()
	defer lock.Unlock()
	if sender == nil {
		// init()関数により初期化が完了する前に、sendLog()が実行された。
		// この状態ではlogmsgを送信することが出来ないため、バッファに蓄積しておく。
		initBuffer = append(initBuffer, logmsg)
	} else {
		if err := sender.SendLog(logmsg); err != nil {
			log.Panicf("failed to sender.Send():err=%s sender=%+v ", err, sender)
		}
		types.RawFuncLogPool.Put(logmsg)
	}
}

func Close() {
	lock.Lock()
	defer lock.Unlock()

	if sender == nil {
		// sender is already closed.
		return
	}

	if err := sender.Close(); err != nil {
		log.Panicf("failed to sender.Close(): err=%s sender=%+v", err, sender)
	}
	sender = nil
}

// tracer.builderパッケージにより編集されたコードから呼び出される関数。
// 全てのログを送信してからプログラムを終了させるために使用する。
func CloseAndExit(code int) {
	Close()
	os.Exit(code)
}

func setOutput() {
	if sender != nil {
		// sender is already opened.
		return
	}

	if CanUseLogServerSender() {
		sender = &RetrySender{
			Sender:        &LogServerSender{},
			MaxRetry:      defaultMaxRetry,
			RetryInterval: defaultRetryInterval,
		}
	} else {
		sender = &RetrySender{
			Sender:        &FileSender{},
			MaxRetry:      defaultMaxRetry,
			RetryInterval: defaultRetryInterval,
		}
	}
	if err := sender.Open(); err != nil {
		log.Panicf("failed to sender.Open(): err=%s sender=%+v", err, sender)
	}
}

func FuncStart() (id types.TxID) {
	id = types.NewTxID()
	sendLog(types.FuncStart, id)
	return
}

func FuncEnd(id types.TxID) {
	sendLog(types.FuncEnd, id)
}

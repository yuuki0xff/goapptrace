package logger

import (
	"errors"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

const (
	defaultMaxRetry      = 50
	defaultRetryInterval = 1 * time.Second
	skips                = 3
	maxStackSize         = 1024

	useCallersFrames = false //@@GAT#FLAG#
)

var (
	ClosedError = errors.New("already closed")

	lock    = sync.Mutex{}
	symbols = logutil.Symbols{
		Writable: true,
		KeepID:   false,
	}
	initBuffer []*logutil.RawFuncLog
	sender     Sender
)

func init() {
	// get all symbols in this process.
	var sd logutil.SymbolsData

	//@@GAT@useNonStandardRuntime@ /*
	/*/
	fname2fileID := func(fname string) logutil.FileID {
		for i, f := range sd.Files {
			if f == fname {
				return logutil.FileID(i)
			}
		}
		sd.Files = append(sd.Files, fname)
		return logutil.FileID(len(sd.Files) - 1)
	}
	runtime.IterateSymbols(
		func(minpc, maxpc uintptr, name string) {
			sd.Mods = append(sd.Mods, logutil.GoModule{
				Name:  name,
				MinPC: minpc,
				MaxPC: maxpc,
			})
		},
		func(pc uintptr, name string) {
			sd.Funcs = append(sd.Funcs, &logutil.GoFunc{
				Entry: pc,
				Name:  name,
			})
		},
		func(pc uintptr, file string, line int32) {
			if line < 0 {
				log.Panicf("invalid line: pc=%d, file=%s, line=%d", pc, file, line)
			}

			sd.Lines = append(sd.Lines, &logutil.GoLine{
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
		sender.SendLog(raw)
	}
	initBuffer = nil
	lock.Unlock()
	symbols.Init()
}

func gid() logutil.GID {
	// get GoroutineID (GID)
	//@@GAT@useNonStandardRuntime@ /*

	// ここはgoapptrace以外の環境でコンパイルしたときに実行される。
	panic("not supported")

	/*/

	// ここは、`goapptrace run`を用いてコンパイルしたときに実行される。
	// runtime.GoID()は、標準のruntimeパッケージ内に存在しない関数である。
	// tracer/builderパッケージによってパッチが当てられた環境でのみ使用可能。
	return logutil.GID(runtime.GoID())

	//*/
}

func sendLog(tag logutil.TagName, id logutil.TxID) {
	logmsg := &logutil.RawFuncLog{}
	logmsg.Tag = tag
	logmsg.Timestamp = logutil.NewTime(time.Now())
	// TODO: goroutine localな変数に、logmsg.Framesで確保するバッファをキャッシュする
	logmsg.Frames = make([]uintptr, 0, maxStackSize)
	logmsg.GID = gid()
	logmsg.TxID = id

	// メモリ確保のオーバーヘッドを削減するために、stack allocateされる固定長配列を使用する。
	// MaxStackSizeを超えている場合、正しいログが取得できない。
	pclen := runtime.Callers(skips, logmsg.Frames[:])
	logmsg.Frames = logmsg.Frames[:pclen]

	// TODO: インライン化やループ展開により、正しくないデータが帰ってくる可能性がある問題を修正する。
	// これらは過去のコードであるが、今後の実装の参考になる可能性があるため、残しておく。
	//
	// symbolsに必要なシンボルを追加とlogmsg.Framesの作成を行う。
	if useCallersFrames {
		// runtime.CallersFrames()を使用する。
		// インライン化やループ展開がされた場合に、行番号や呼び出し元関数の調整を行うことができる。
		// しかし、オーバーヘッドが大きくなる。
		// コンパイラの最適化を簡単に無効化できない場合に使用することを推薦する。
	} else {
		// runtime.FuncForPC()を使用する。
		// runtime.CallersFrames()を使用するよりもオーバーヘッドが少ない。
		// ただし、最適化が行われると呼び出し元の判定が狂ってしまう。
		// これを使用するときは、*最適化を無効*にしてコンパイルすること。
	}

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

func FuncStart() (id logutil.TxID) {
	id = logutil.NewTxID()
	sendLog(logutil.FuncStart, id)
	return
}

func FuncEnd(id logutil.TxID) {
	sendLog(logutil.FuncEnd, id)
}

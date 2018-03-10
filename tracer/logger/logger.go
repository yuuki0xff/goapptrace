package logger

import (
	"errors"
	"log"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

const (
	defaultMaxRetry      = 50
	defaultRetryInterval = 1 * time.Second
	skips                = 3
	backtraceSize        = 1 << 16 // about 64KiB
	maxStackSize         = 1024

	useCallersFrames      = false //@@GAT#FLAG#
	useNonStandardRuntime = false //@@GAT#FLAG#
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

	// stack traceからGID(Goroutine ID)を取得するための正規表現
	gidRegExp = regexp.MustCompile(`^goroutine (\d+)`)
)

func init() {
	if useNonStandardRuntime {
		// get all symbols in this process.
		// TODO: call to some method in Symbols. Need to implement it before write this function.
		//@@GAT@useNonStandardRuntime@ runtime.IterateSymbols(
		//@@GAT@useNonStandardRuntime@ 	nil,
		//@@GAT@useNonStandardRuntime@ 	nil,
		//@@GAT@useNonStandardRuntime@ 	nil,
		//@@GAT@useNonStandardRuntime@ )

		lock.Lock()
		setOutput()
		if sender == nil {
			log.Panicln("sender is nil")
		}
		// TODO: send symbols.
		// TODO: send buffered logs on initBuffer.
		initBuffer = nil
		lock.Unlock()
	}
	symbols.Init()
}

func gid() logutil.GID {
	// get GoroutineID (GID)
	var id logutil.GID
	if useNonStandardRuntime {
		// runtime.GoID()は、標準のruntimeパッケージ内に存在しない関数である。
		// tracer/builderパッケージによってパッチが当てられた環境でのみ使用可能。

		//@@GAT@useNonStandardRuntime@ id = logutil.GID(runtime.GoID())
	} else {
		var buf [backtraceSize]byte
		runtime.Stack(buf[:], false) // First line is "goroutine xxx [running]"
		matches := gidRegExp.FindSubmatch(buf[:])
		gid, err := strconv.ParseInt(string(matches[1]), 10, 64)
		if err != nil {
			log.Panic(err)
		}
		id = logutil.GID(gid)
	}
	return id
}

func sendLog(tag logutil.TagName, id logutil.TxID) {
	// TODO: 初期化前だったときの処理を追加する

	shouldSendDiff := false
	diff := &logutil.SymbolsData{}

	logmsg := &logutil.RawFuncLog{}
	logmsg.Tag = tag
	logmsg.Timestamp = logutil.NewTime(time.Now())
	logmsg.Frames = make([]logutil.FuncStatusID, 0, maxStackSize)
	logmsg.GID = gid()
	logmsg.TxID = id

	// TODO: goroutine localな変数に、pcsBuffをキャッシュする
	// メモリ確保のオーバーヘッドを削減するために、stack allocateされる固定長配列を使用する。
	// MaxStackSizeを超えている場合、正しいログが取得できない。
	var pcsBuff [maxStackSize]uintptr
	pclen := runtime.Callers(skips, pcsBuff[:])
	pcs := pcsBuff[:pclen]

	// TODO: PCsだけをサーバに送信。シンボル解決は事後処理する方式にする。
	// TODO: 全シンボルはプロセス初期化時にサーバに送信する。
	// symbolsに必要なシンボルを追加とlogmsg.Framesの作成を行う。
	if useCallersFrames {
		// runtime.CallersFrames()を使用する。
		// インライン化やループ展開がされた場合に、行番号や呼び出し元関数の調整を行うことができる。
		// しかし、オーバーヘッドが大きくなる。
		// コンパイラの最適化を簡単に無効化できない場合に使用することを推薦する。

		frames := runtime.CallersFrames(pcs)
		for {
			frame, more := frames.Next()
			if !more {
				break
			}
			fsid, ok := symbols.FuncStatusIDFromPC(frame.PC)
			if !ok {
				// SLOW PATH
				shouldSendDiff = true

				fid, ok := symbols.FuncIDFromName(frame.Function)
				if !ok {
					// FuncSymbolが未登録なので、追加する。
					var funcWasAdded bool
					fid, funcWasAdded = symbols.AddFunc(&logutil.GoFunc{
						Name:  frame.Function,
						File:  frame.File,
						Entry: frame.Entry,
					})
					if funcWasAdded {
						f := &logutil.GoFunc{}
						*f, _ = symbols.Func(fid)
						diff.Funcs = append(diff.Funcs, f)
					}
				}

				// FuncSymbolを追加する。
				var funcStatusWasAdded bool
				fsid, funcStatusWasAdded = symbols.AddFuncStatus(&logutil.FuncStatus{
					Func: fid,
					Line: uint64(frame.Line),
					PC:   frame.PC,
				})
				if funcStatusWasAdded {
					f := &logutil.FuncStatus{}
					*f, _ = symbols.FuncStatus(fsid)
					diff.FuncStatus = append(diff.FuncStatus, f)
				}
			}
			logmsg.Frames = append(logmsg.Frames, fsid)
		}
	} else {
		// runtime.FuncForPC()を使用する。
		// runtime.CallersFrames()を使用するよりもオーバーヘッドが少ない。
		// ただし、最適化が行われると呼び出し元の判定が狂ってしまう。
		// これを使用するときは、*最適化を無効*にしてコンパイルすること。

		for _, pc := range pcs {
			fsid, ok := symbols.FuncStatusIDFromPC(pc)
			if !ok {
				// SLOW PATH
				shouldSendDiff = true

				f := runtime.FuncForPC(pc)
				fid, ok := symbols.FuncIDFromName(f.Name())
				if !ok {
					// FuncSymbolが未登録なので、追加する。
					var funcWasAdded bool
					file, _ := f.FileLine(f.Entry())
					fid, funcWasAdded = symbols.AddFunc(&logutil.GoFunc{
						Name:  f.Name(),
						File:  file,
						Entry: f.Entry(),
					})
					if funcWasAdded {
						f := &logutil.GoFunc{}
						*f, _ = symbols.Func(fid)
						diff.Funcs = append(diff.Funcs, f)
					}
				}

				// FuncSymbolを追加する。
				var funcStatusWasAdded bool
				_, line := f.FileLine(pc)
				fsid, funcStatusWasAdded = symbols.AddFuncStatus(&logutil.FuncStatus{
					Func: fid,
					Line: uint64(line),
					PC:   pc,
				})
				if funcStatusWasAdded {
					f := &logutil.FuncStatus{}
					*f, _ = symbols.FuncStatus(fsid)
					diff.FuncStatus = append(diff.FuncStatus, f)
				}
			}
			logmsg.Frames = append(logmsg.Frames, fsid)
		}
	}

	if !shouldSendDiff {
		diff = nil
	}

	// TODO: 排他ロックを取って、sendBufferに直接書き込む。
	// CPUのキャッシュに乗っているうちにシリアライズしたほうがよい。
	lock.Lock()
	defer lock.Unlock()
	if sender == nil {
		setOutput()
	}
	if err := sender.Send(diff, logmsg); err != nil {
		log.Panicf("failed to sender.Send():err=%s sender=%+v ", err, sender)
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

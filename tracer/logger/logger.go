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

	"github.com/bouk/monkey"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

const (
	defaultMaxRetry      = 50
	defaultRetryInterval = 1 * time.Second
	skips                = 3
	backtraceSize        = 1 << 16 // about 64KiB
)

var (
	MaxStackSize = 1024
	ClosedError  = errors.New("already closed")

	lock       = sync.Mutex{}
	symbols    = logutil.Symbols{}
	patchGuard *monkey.PatchGuard
	sender     Sender
)

func init() {
	symbols.Init(true, false)

	// os.Exitにフックを仕掛ける
	// TODO: don't work!
	patchGuard = monkey.Patch(os.Exit, func(code int) {
		patchGuard.Unpatch()
		defer patchGuard.Restore()

		// close a file or client before exit.
		Close()
		os.Exit(code)
	})
}

func sendLog(tag logutil.TagName, id logutil.TxID) {
	var newSymbols *logutil.Symbols

	logmsg := &logutil.RawFuncLog{}
	logmsg.Timestamp = logutil.NewTime(time.Now())
	logmsg.Tag = tag
	logmsg.Frames = make([]logutil.FuncStatusID, 0, MaxStackSize)
	logmsg.TxID = id

	pc := make([]uintptr, MaxStackSize)
	pclen := runtime.Callers(skips, pc)
	pc = pc[:pclen]

	// TODO: improve performance
	//       CallersFrames()で全てのフレームをframeを取得 & symbolsに追加するのは非効率。
	//       symbols.FindFunStatusIDByPC(pc)をまず試す。
	//       idを取得できなかったら、runtime.FuncForPC()でframeが取得後、symbolsに追加する。
	frames := runtime.CallersFrames(pc)
	for {
		frame, more := frames.Next()
		if !more {
			break
		}

		funcID, added1 := symbols.AddFunc(&logutil.FuncSymbol{
			Name:  frame.Function,
			File:  frame.File,
			Entry: frame.Entry,
		})
		funcStatusID, added2 := symbols.AddFuncStatus(&logutil.FuncStatus{
			Func: funcID,
			Line: uint64(frame.Line),
			PC:   frame.PC,
		})
		logmsg.Frames = append(logmsg.Frames, funcStatusID)

		if added1 || added2 {
			if newSymbols == nil {
				// prepare newSymbols
				newSymbols = &logutil.Symbols{}
				newSymbols.Init(true, true)
			}

			if added1 {
				newSymbols.Funcs = append(newSymbols.Funcs, symbols.Funcs[funcID])
			}
			if added2 {
				newSymbols.FuncStatus = append(newSymbols.FuncStatus, symbols.FuncStatus[funcStatusID])
			}
		}
	}

	// get GoroutineID (GID)
	buf := make([]byte, backtraceSize)
	runtime.Stack(buf, false) // First line is "goroutine xxx [running]"
	re := regexp.MustCompile(`^goroutine (\d+)`)
	matches := re.FindSubmatch(buf)
	gid, err := strconv.ParseInt(string(matches[1]), 10, 64)
	if err != nil {
		log.Panic(err)
	}
	logmsg.GID = logutil.GID(gid)

	lock.Lock()
	defer lock.Unlock()
	if sender == nil {
		setOutput()
	}
	if err := sender.Send(newSymbols, logmsg); err != nil {
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

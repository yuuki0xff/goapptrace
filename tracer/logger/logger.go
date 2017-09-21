package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"errors"

	"regexp"
	"strconv"

	"math/rand"

	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/protocol"
)

const (
	skips         = 3
	backtraceSize = 1 << 16 // about 64KiB
)

var (
	MaxStackSize = 1024
	OutputFile   *os.File
	Client       *protocol.Client

	lock           = sync.Mutex{}
	symbols        = logutil.Symbols{}
	symbolResolver = logutil.SymbolResolver{}
)

func init() {
	symbols.Init()
	symbolResolver.Init(&symbols)
}

func sendLog(tag string, id logutil.TxID) {
	var newSymbols *logutil.Symbols

	logmsg := logutil.RawFuncLogNew{}
	logmsg.Timestamp = time.Now().Unix()
	logmsg.Tag = tag
	logmsg.Frames = make([]logutil.FuncStatusID, 0, MaxStackSize)
	logmsg.TxID = id

	pc := make([]uintptr, MaxStackSize)
	pclen := runtime.Callers(skips, pc)
	pc = pc[:pclen]

	frames := runtime.CallersFrames(pc)
	for {
		frame, more := frames.Next()
		if more == false {
			break
		}

		funcID, added1 := symbolResolver.AddFunc(&logutil.FuncSymbol{
			Name:  frame.Function,
			File:  frame.File,
			Entry: frame.Entry,
		})
		funcStatusID, added2 := symbolResolver.AddFuncStatus(&logutil.FuncStatus{
			Func: funcID,
			Line: uint64(frame.Line),
			PC:   frame.PC,
		})
		logmsg.Frames = append(logmsg.Frames, funcStatusID)

		if added1 || added2 {
			// prepare newSymbols
			newSymbols = &logutil.Symbols{}
			newSymbols.Init()

			if added1 {
				newSymbols.Funcs = append(newSymbols.Funcs, symbols.Funcs[funcID])
			}
			if added2 {
				newSymbols.FuncStatus = append(newSymbols.FuncStatus, symbols.FuncStatus[funcStatusID])
			}
		}
	}

	buf := make([]byte, backtraceSize)
	runtime.Stack(buf, false) // First line is "goroutine xxx [running]"
	re := regexp.MustCompile(`^goroutine (\d+)`)
	matches := re.FindSubmatch(buf)
	gid, err := strconv.ParseInt(string(matches[1]), 10, 64)
	if err != nil {
		panic(err)
	}
	logmsg.GID = logutil.GID(gid)

	lock.Lock()
	defer lock.Unlock()
	if OutputFile == nil && Client == nil {
		setOutput()
	}

	if OutputFile != nil {
		// write symbols to file
		if newSymbols != nil {
			err := json.NewEncoder(OutputFile).Encode(newSymbols)
			if err != nil {
				panic(err)
			}
		}
		_, err = OutputFile.Write([]byte("\n"))
		if err != nil {
			panic(err)
		}

		// write backtrace to file
		err := json.NewEncoder(OutputFile).Encode(&logmsg)
		if err != nil {
			panic(err)
		}
		_, err = OutputFile.Write([]byte("\n"))
		if err != nil {
			panic(err)
		}
	} else if Client != nil {
		// send binary log to log server
		if newSymbols != nil {
			Client.Send(protocol.SymbolsMsg, newSymbols)
		}
		Client.Send(protocol.RawFuncLogMsg, logmsg)
	} else {
		panic(errors.New("here is unreachable, but reached"))
	}
}

func Close() {
	if OutputFile != nil {
		OutputFile.Close()
		OutputFile = nil
	} else if Client != nil {
		Client.Close()
		Client = nil
	} else {
		panic(errors.New("here is unreachable, but reached"))
	}
}

func setOutput() {
	pid := os.Getpid()
	url, ok := os.LookupEnv(info.DEFAULT_LOGSRV_ENV)
	if ok {
		// use socket
		Client = &protocol.Client{
			Addr: url,
			Handler: protocol.ClientHandler{
				Connected:    func() {},
				Disconnected: func() {},
				Error: func(err error) {
					fmt.Println("Client ERROR:", err.Error())
				},
				StartTrace: func(args *protocol.StartTraceCmdArgs) {},
				StopTrace:  func(args *protocol.StopTraceCmdArgs) {},
			},
			AppName: "TODO", // TODO
			Version: info.VERSION,
			Secret:  "secret",
		}
		if err := Client.Connect(); err != nil {
			panic(err)
		}
	} else {
		// use log file
		prefix, ok := os.LookupEnv(info.DEFAULT_LOGFILE_ENV)
		if !ok {
			prefix = info.DEFAULT_LOGFILE_PREFIX
		}
		fpath := fmt.Sprintf("%s.%d.log", prefix, pid)
		file, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		OutputFile = file
	}
}

func FuncStart() (id logutil.TxID) {
	id = logutil.TxID(rand.Int63())
	sendLog("funcStart", id)
	return
}

func FuncEnd(id logutil.TxID) {
	sendLog("funcEnd", id)
}

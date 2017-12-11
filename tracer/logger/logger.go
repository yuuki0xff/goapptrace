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

	"log"

	"github.com/bouk/monkey"
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

	lock          = sync.Mutex{}
	symbols       = logutil.Symbols{}
	symbolsEditor = logutil.SymbolsEditor{}
	patchGuard    *monkey.PatchGuard
)

func init() {
	symbols.Init()
	symbolsEditor.Init(&symbols)

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

func sendLog(tag string, id logutil.TxID) {
	var newSymbols *logutil.Symbols

	logmsg := &logutil.RawFuncLogNew{}
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

		funcID, added1 := symbolsEditor.AddFunc(&logutil.FuncSymbol{
			Name:  frame.Function,
			File:  frame.File,
			Entry: frame.Entry,
		})
		funcStatusID, added2 := symbolsEditor.AddFuncStatus(&logutil.FuncStatus{
			Func: funcID,
			Line: uint64(frame.Line),
			PC:   frame.PC,
		})
		logmsg.Frames = append(logmsg.Frames, funcStatusID)

		if added1 || added2 {
			if newSymbols == nil {
				// prepare newSymbols
				newSymbols = &logutil.Symbols{}
				newSymbols.Init()
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
	if OutputFile == nil && Client == nil {
		setOutput()
	}

	if OutputFile != nil {
		// write symbols to file
		if newSymbols != nil {
			err := json.NewEncoder(OutputFile).Encode(newSymbols)
			if err != nil {
				log.Panic(err)
			}
		}
		_, err = OutputFile.Write([]byte("\n"))
		if err != nil {
			log.Panic(err)
		}

		// write backtrace to file
		err := json.NewEncoder(OutputFile).Encode(&logmsg)
		if err != nil {
			log.Panic(err)
		}
		_, err = OutputFile.Write([]byte("\n"))
		if err != nil {
			log.Panic(err)
		}
	} else if Client != nil {
		// send binary log to log server
		if newSymbols != nil {
			if err := Client.Send(&protocol.SymbolPacket{newSymbols}); err != nil {
				// TODO: try to reconnect
				log.Panic(err)
			}
		}
		if err := Client.Send(&protocol.RawFuncLogNewPacket{logmsg}); err != nil {
			// TODO: try to reconnect
			log.Panic(err)
		}
	} else {
		log.Panic(errors.New("here is unreachable, but reached"))
	}
}

func Close() {
	if OutputFile != nil {
		if err := OutputFile.Close(); err != nil {
			log.Panic(err)
		}
		OutputFile = nil
	} else if Client != nil {
		if err := Client.Close(); err != nil {
			log.Panic(err)
		}
		Client = nil
	} else {
		// ignore double-closing
		//log.Panic(errors.New("here is unreachable, but reached"))
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
				StartTrace: func(args *protocol.StartTraceCmdPacket) {},
				StopTrace:  func(args *protocol.StopTraceCmdPacket) {},
			},
			AppName: "TODO", // TODO
			Secret:  "secret",
		}
		Client.Init()
		go func() {
			if err := Client.Serve(); err != nil {
				log.Panic(err)
			}
		}()
		Client.WaitNegotiation()
	} else {
		// use log file
		prefix, ok := os.LookupEnv(info.DEFAULT_LOGFILE_ENV)
		if !ok {
			prefix = info.DEFAULT_LOGFILE_PREFIX
		}
		fpath := fmt.Sprintf("%s.%d.log", prefix, pid)
		file, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Panic(err)
		}
		OutputFile = file
	}
}

func FuncStart() (id logutil.TxID) {
	id = logutil.NewTxID()
	sendLog("funcStart", id)
	return
}

func FuncEnd(id logutil.TxID) {
	sendLog("funcEnd", id)
}

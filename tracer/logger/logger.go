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

	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/log"
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

	lock = sync.Mutex{}
)

func sendLog(tag string) {
	logmsg := log.RawLog{}
	logmsg.Timestamp = time.Now().Unix()
	logmsg.Tag = tag
	logmsg.Frames = make([]runtime.Frame, 0, MaxStackSize)

	pc := make([]uintptr, MaxStackSize)
	pclen := runtime.Callers(skips, pc)
	pc = pc[:pclen]

	frames := runtime.CallersFrames(pc)
	for {
		frame, more := frames.Next()
		if more == false {
			break
		}

		logmsg.Frames = append(logmsg.Frames, frame)
	}

	buf := make([]byte, backtraceSize)
	runtime.Stack(buf, false) // First line is "goroutine xxx [running]"
	re := regexp.MustCompile(`^goroutine (\d+)`)
	matches := re.FindSubmatch(buf)
	gid, err := strconv.ParseInt(string(matches[1]), 10, 64)
	if err != nil {
		panic(err)
	}
	logmsg.GID = log.GID(gid)

	lock.Lock()
	defer lock.Unlock()
	if OutputFile == nil && Client == nil {
		setOutput()
	}

	if OutputFile != nil {
		js, err := json.Marshal(logmsg)
		if err != nil {
			panic(err)
		}

		// write to file
		_, err = OutputFile.Write(js)
		if err != nil {
			panic(err)
		}
		_, err = OutputFile.Write([]byte("\n"))
		if err != nil {
			panic(err)
		}
	} else if Client != nil {
		// TODO: send binary log to log server
		// TODO: ログのフォーマットがprotocolと統一されていない。txIDの導入などを行う
		//Client.Send(protocol.FuncLogMsg, struct{}{})
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
				Error:        func(err error) {},
				StartTrace:   func(args *protocol.StartTraceCmdArgs) {},
				StopTrace:    func(args *protocol.StopTraceCmdArgs) {},
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

func FuncStart() {
	sendLog("funcStart")
}

func FuncEnd() {
	sendLog("funcEnd")
}

package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/yuuki0xff/goapptrace/info"
)

const (
	skips = 3
)

var (
	MaxStackSize = 1024
	OutputFile   *os.File

	lock = sync.Mutex{}
)

type LogMessage struct {
	Timestamp int64           `json:"timestamp"`
	Tag       string          `json:"tag"`
	Frames    []runtime.Frame `json:"frames"`
}

func sendLog(tag string) {
	logmsg := LogMessage{}
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
	js, err := json.Marshal(logmsg)
	if err != nil {
		panic(err)
	}

	lock.Lock()
	defer lock.Unlock()
	if OutputFile == nil {
		pid := os.Getpid()
		prefix, ok := os.LookupEnv(info.DEFAULT_LOGFILE_ENV)
		if !ok {
			prefix = info.DEFAULT_LOGFILE_PREFIX
		}
		fpath := fmt.Sprintf("%s.%d.log", prefix, pid)
		OutputFile, err = os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
	}
	_, err = OutputFile.Write(js)
	if err != nil {
		panic(err)
	}
	_, err = OutputFile.Write([]byte("\n"))
	if err != nil {
		panic(err)
	}
}

func FuncStart() {
	sendLog("funcStart")
}

func FuncEnd() {
	sendLog("funcEnd")
}

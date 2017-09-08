package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"net"
	"strings"

	"errors"

	"regexp"
	"strconv"

	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/log"
)

const (
	skips         = 3
	backtraceSize = 1 << 16 // about 64KiB
)

var (
	MaxStackSize = 1024
	OutputFile   *os.File
	OutputSocket net.Conn

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
	if OutputFile == nil && OutputSocket == nil {
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
	} else if OutputSocket != nil {
		// TODO: send binary log to log server
	} else {
		panic(errors.New("here is unreachable, but reached"))
	}
}

func setOutput() {
	pid := os.Getpid()
	url, ok := os.LookupEnv(info.DEFAULT_LOGSRV_ENV)
	if ok {
		// use socket
		result := strings.SplitN(url, "://", 2)
		proto := result[0]
		hostport := result[1]

		conn, err := net.Dial(proto, hostport)
		if err != nil {
			panic(err)
		}
		OutputSocket = conn
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

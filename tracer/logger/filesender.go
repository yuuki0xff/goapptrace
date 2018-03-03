package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/yuuki0xff/goapptrace/info"
	"github.com/yuuki0xff/goapptrace/tracer/logutil"
	"github.com/yuuki0xff/goapptrace/tracer/storage"
)

// FileSender writes Symbols and FuncLog to log file.
type FileSender struct {
	w *storage.CompactLogWriter
}

// open log file.
func (f *FileSender) Open() error {
	clog := storage.CompactLog{
		File: storage.File(f.logFilePath()),
	}
	f.w = clog.Writer()
	return f.w.Open()
}

// close log file.
// これ移行はSendできない。
func (f *FileSender) Close() error {
	if f.w == nil {
		return ClosedError
	}
	err := f.w.Close()
	f.w = nil
	return err
}

// write Symbols and RawFuncLog to the log file.
func (f *FileSender) Send(diff *logutil.SymbolsData, funclog *logutil.RawFuncLog) error {
	if f.w == nil {
		return ClosedError
	}
	return f.w.Write(diff, funclog)
}

// returns absolute path of log file.
func (f *FileSender) logFilePath() string {
	pid := os.Getpid()
	prefix, ok := os.LookupEnv(info.DEFAULT_LOGFILE_ENV)
	if !ok {
		prefix = info.DEFAULT_LOGFILE_PREFIX
	}
	relativePath := fmt.Sprintf("%s.%d.log.gz", prefix, pid)
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		log.Panic(err)
	}
	return absPath
}
